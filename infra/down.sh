#!/usr/bin/env bash
# down.sh saves the certificate one last time, in case certbot renewed it,
# then deletes everything Terraform created.
set -euo pipefail
cd "$(dirname "$0")"

if API=$(terraform output -raw api_ip 2> /dev/null) && [ -n "$API" ]; then
  DOMAIN=$(terraform output -raw domain)
  KEY=$(terraform output -raw ssh_key)
  CERT="secrets/letsencrypt-$DOMAIN.tar.gz"
  if ssh -i "$KEY" -o UserKnownHostsFile=secrets/known_hosts -o ConnectTimeout=5 "ubuntu@$API" \
    'sudo tar czf - -C / etc/letsencrypt' > "$CERT.tmp" 2> /dev/null && [ -s "$CERT.tmp" ]; then
    mv "$CERT.tmp" "$CERT"
    chmod 600 "$CERT"
    echo "saved the certificate to infra/$CERT"
  else
    rm -f "$CERT.tmp"
  fi
fi

terraform destroy -input=false -auto-approve
rm -f secrets/known_hosts
