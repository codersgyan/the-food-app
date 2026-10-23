#!/usr/bin/env bash
# setup.sh does steps 2 to 5 of deploy/README.md on a fresh Ubuntu server, in
# the same order, and prints each step as it goes. It's safe to run again: a
# step that's already done is skipped.
#
# Run it on the server, from the repo folder:
#
#     DOMAIN=food-api.sys0.in EMAIL=you@example.com ./deploy/setup.sh
#
# SEED=small (the default) adds the sample data, SEED=big adds the dataset
# from the indexes lesson, and SEED=none adds nothing. Seeding empties every
# table first, so it only happens when the database has no restaurants yet.
set -euo pipefail

DOMAIN=${DOMAIN:?set DOMAIN, like food-api.sys0.in}
EMAIL=${EMAIL:?set EMAIL, for the certificate}
SEED=${SEED:-small}
cd "$(dirname "$0")/.."
step() { printf '\n=== %s\n' "$*"; }
export DEBIAN_FRONTEND=noninteractive

step "2. Install Docker from Docker's own apt repository"
if ! command -v docker > /dev/null; then
  sudo apt-get update -qq
  sudo apt-get install -y -qq ca-certificates curl
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc
  sudo tee /etc/apt/sources.list.d/docker.sources > /dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
  sudo apt-get update -qq
  sudo apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi
sudo docker compose version

step "3. Make the settings file"
if ! sudo test -f /etc/food-api.env; then
  sudo tee /etc/food-api.env > /dev/null <<EOF
JWT_SECRET=$(openssl rand -hex 32)
PAYMENT_WEBHOOK_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
EOF
  sudo chmod 600 /etc/food-api.env
  echo "made /etc/food-api.env with new secrets"
else
  echo "/etc/food-api.env already exists, keeping it"
fi

step "4. Start the API server and Postgres"
COMPOSE="sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env"
$COMPOSE up -d --build
sudo docker builder prune -f > /dev/null
$COMPOSE run --rm api migrate up
RESTAURANTS=$($COMPOSE exec -T postgres psql -U food -d food -Atc 'SELECT count(*) FROM restaurants')
if [ "$RESTAURANTS" = 0 ]; then
  case $SEED in
    small) $COMPOSE run --rm api seed ;;
    big) $COMPOSE run --rm api seed -restaurants 20000 -dishes 100 -customers 10000 ;;
    none) echo "not seeding" ;;
    *) echo "SEED must be small, big or none" >&2; exit 1 ;;
  esac
else
  echo "the database already has $RESTAURANTS restaurants, not seeding"
fi
curl -sf -o /dev/null 127.0.0.1:8080/v1/restaurants/1/menu && echo "the API server answers on 127.0.0.1:8080"

step "5. Get a certificate, then start nginx"
sudo apt-get install -y -qq nginx certbot python3-certbot-nginx bind9-dnsutils

# Ubuntu's nginx allows 768 connections per worker. Our load test showed that
# runs out at twice the dinner peak, so allow more, and let each worker open
# enough files for them.
sudo sed -i 's/worker_connections 768;/worker_connections 4096;/' /etc/nginx/nginx.conf
grep -q '^worker_rlimit_nofile' /etc/nginx/nginx.conf ||
  sudo sed -i '/^worker_processes/a worker_rlimit_nofile 8192;' /etc/nginx/nginx.conf

# Let's Encrypt checks the domain over port 80, so the domain has to point here first.
MY_IP=$(curl -s https://checkip.amazonaws.com)
for _ in $(seq 1 60); do
  [ "$(dig +short "$DOMAIN" @1.1.1.1 | tail -1)" = "$MY_IP" ] && break
  echo "waiting for $DOMAIN to point to $MY_IP"
  sleep 10
done

if sudo test -d "/etc/letsencrypt/live/$DOMAIN"; then
  echo "a certificate for $DOMAIN is already here, keeping it"
else
  sudo certbot certonly --nginx -d "$DOMAIN" --deploy-hook "systemctl reload nginx" \
    --agree-tos -m "$EMAIL" --no-eff-email --non-interactive
fi

sed "s/food-api\.sys0\.in/$DOMAIN/g" deploy/nginx/food-api.sys0.in.conf |
  sudo tee "/etc/nginx/sites-available/$DOMAIN.conf" > /dev/null
sudo ln -sf "/etc/nginx/sites-available/$DOMAIN.conf" /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

step "Check it"
curl -sf -o /dev/null "https://$DOMAIN/v1/restaurants/1/menu" && echo "https://$DOMAIN answers"
