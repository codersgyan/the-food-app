#!/usr/bin/env bash
# collect-env.sh prints everything about the API server that a test result
# depends on: the machine, the versions, the settings and the data. Run it on
# the API server, from the repo folder. It never prints a secret.
set -u
C="sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env"
psql() { $C exec -T postgres psql -U food -d food -At -c "$1"; }

meta() {
  local token
  token=$(curl -s -m 2 -X PUT http://169.254.169.254/latest/api/token -H "X-aws-ec2-metadata-token-ttl-seconds: 60")
  curl -s -m 2 -H "X-aws-ec2-metadata-token: $token" "http://169.254.169.254/latest/meta-data/$1"
}

echo "== machine"
echo "instance type:      $(meta instance-type)"
echo "availability zone:  $(meta placement/availability-zone)"
echo "vCPUs:              $(nproc)"
echo "memory:             $(free -m | awk '/Mem:/ {print $2}') MB"
echo "os:                 $(. /etc/os-release && echo "$PRETTY_NAME")"
echo "kernel:             $(uname -r)"
echo "arch:               $(uname -m)"

echo
echo "== software"
echo "git commit:         $(git rev-parse --short HEAD) ($(git log -1 --format=%s))"
echo "docker:             $(sudo docker version --format '{{.Server.Version}}')"
echo "compose:            $(sudo docker compose version --short)"
echo "nginx:              $(nginx -v 2>&1 | cut -d/ -f2)"
echo "postgres:           $(psql 'SHOW server_version')"
echo "api image:          $(sudo docker inspect -f '{{.Image}}' the-food-app-prod-api-1 | cut -c1-19)"

echo
echo "== settings"
echo "api settings set in /etc/food-api.env (names only): $(sudo cut -d= -f1 /etc/food-api.env | tr '\n' ' ')"
echo "database pool:      pgxpool default, max(4, CPUs) = $(( $(nproc) > 4 ? $(nproc) : 4 )) connections"
for s in max_connections shared_buffers effective_cache_size work_mem; do
  printf '%-20s%s\n' "postgres $s:" "$(psql "SHOW $s")"
done
echo "nginx workers:      $(grep -E '^\s*worker_processes' /etc/nginx/nginx.conf | tr -d ';' | awk '{print $2}') (auto = one per CPU)"
echo "nginx connections:  $(grep -E '^\s*worker_connections' /etc/nginx/nginx.conf | tr -d ';' | awk '{print $2}') per worker"

echo
echo "== data"
for t in restaurants dishes users addresses orders order_items; do
  printf '%-20s%s rows\n' "$t:" "$(psql "SELECT count(*) FROM $t")"
done
echo "database size:      $(psql "SELECT pg_size_pretty(pg_database_size('food'))")"
echo "indexes:            $(psql "SELECT string_agg(indexname, ', ' ORDER BY indexname) FROM pg_indexes WHERE schemaname = 'public'")"

echo
echo "== nginx config (/etc/nginx/sites-enabled/food-api.sys0.in.conf)"
cat /etc/nginx/sites-enabled/food-api.sys0.in.conf
