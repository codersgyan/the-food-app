#!/usr/bin/env bash
# up.sh creates the two servers with Terraform, points the domain at the API
# server, sets the API server up with deploy/setup.sh, and installs k6 on the
# test runner. See infra/README.md.
set -euo pipefail
cd "$(dirname "$0")"

: "${EMAIL:?set EMAIL, the address that gets emails about the certificate}"
SEED=${SEED:-big}
REPO=${REPO:-https://github.com/codersgyan/the-food-app.git}
BRANCH=${BRANCH:-main}

mkdir -p secrets
rm -f secrets/known_hosts # new servers have new host keys

terraform init -input=false > /dev/null
terraform apply -input=false -auto-approve

API=$(terraform output -raw api_ip)
RUNNER=$(terraform output -raw runner_ip)
DOMAIN=$(terraform output -raw domain)
KEY=$(terraform output -raw ssh_key)
SSH_OPTS=(-i "$KEY" -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=secrets/known_hosts -o ConnectTimeout=5)
CERT="secrets/letsencrypt-$DOMAIN.tar.gz"

wait_for() {
  echo "waiting for $1 to boot"
  until ssh "${SSH_OPTS[@]}" "ubuntu@$1" true 2> /dev/null; do sleep 5; done
  ssh "${SSH_OPTS[@]}" "ubuntu@$1" 'cloud-init status --wait > /dev/null'
}
wait_for "$API"
wait_for "$RUNNER"

# Reuse the certificate from last time, so Let's Encrypt doesn't have to issue
# a new one for every new server. certbot renews it when it gets old.
if [ -f "$CERT" ]; then
  echo "restoring the certificate for $DOMAIN"
  scp -q "${SSH_OPTS[@]}" "$CERT" "ubuntu@$API:/tmp/letsencrypt.tar.gz"
  ssh "${SSH_OPTS[@]}" "ubuntu@$API" 'sudo tar xzf /tmp/letsencrypt.tar.gz -C / && rm /tmp/letsencrypt.tar.gz'
fi

ssh "${SSH_OPTS[@]}" "ubuntu@$API" "{ [ -d the-food-app ] || git clone -q -b $BRANCH $REPO the-food-app; } && cd the-food-app &&
  DOMAIN=$DOMAIN EMAIL=$EMAIL SEED=$SEED ./deploy/setup.sh"

echo "saving the certificate to infra/$CERT"
ssh "${SSH_OPTS[@]}" "ubuntu@$API" 'sudo tar czf - -C / etc/letsencrypt' > "$CERT.tmp"
mv "$CERT.tmp" "$CERT"
chmod 600 "$CERT"

echo "setting up the test runner"
ssh "${SSH_OPTS[@]}" "ubuntu@$RUNNER" 'bash -s' < ../loadtest/setup-runner.sh

cat << EOF

Ready: https://$DOMAIN
  API server:  ssh -i infra/$KEY ubuntu@$API
  test runner: ssh -i infra/$KEY ubuntu@$RUNNER

Run a load test with: make cloud-loadtest NAME=customers-only STEPS="1 2 3 5 10"
Delete everything with: make cloud-down
EOF
