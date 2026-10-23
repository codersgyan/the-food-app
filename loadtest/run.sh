#!/usr/bin/env bash
# run.sh runs one load test from your laptop, on two servers: the API server,
# and a separate test runner that sends the traffic. It runs one step for each
# multiplier, and stops early when the server can't keep up. Everything it
# measures goes into results/<date>-<name>/.
#
#     export API_HOST=3.111.144.254 RUNNER_HOST=3.108.191.183 SSH_KEY=~/.ssh/pg-load-test.pem
#     ./run.sh customers-only "1 2 3 5 10"
#     TABLETS=20000 TABLET_INTERVAL=10 ./run.sh with-tablets "1 2 3 5 10"
#
# The runner needs k6 and the API server needs the app from deploy/README.md,
# seeded with: run --rm api seed -restaurants 20000 -dishes 100 -customers 10000
set -euo pipefail

NAME=${1:?usage: run.sh <name> "<multipliers>"}
MULTS=${2:?usage: run.sh <name> "<multipliers>"}
: "${API_HOST:?set API_HOST}" "${RUNNER_HOST:?set RUNNER_HOST}" "${SSH_KEY:?set SSH_KEY}"
BASE_URL=${BASE_URL:-https://food-api.sys0.in}
TABLETS=${TABLETS:-0}
TABLET_INTERVAL=${TABLET_INTERVAL:-10}
DURATION=${DURATION:-3m}
REST=${REST:-30} # seconds between steps, so one step's tail doesn't spill into the next

HERE=$(cd "$(dirname "$0")" && pwd)
OUT="$HERE/results/$(date -u +%Y-%m-%d)-$NAME"
mkdir -p "$OUT"

# KNOWN_HOSTS lets make cloud-loadtest keep the servers' host keys in infra/secrets/.
SSH_OPTS=(-i "$SSH_KEY" -o ServerAliveInterval=30 -o StrictHostKeyChecking=accept-new
  -o UserKnownHostsFile="${KNOWN_HOSTS:-$HOME/.ssh/known_hosts}")
api() { ssh "${SSH_OPTS[@]}" "ubuntu@$API_HOST" "$@"; }
runner() { ssh "${SSH_OPTS[@]}" "ubuntu@$RUNNER_HOST" "$@"; }
COMPOSE='cd the-food-app && sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env'
pg_stats() { api "$COMPOSE exec -T postgres psql -U food -d food -At" < "$HERE/pg-stats.sql" > "$1"; }

echo "copying the scripts"
scp -q "${SSH_OPTS[@]}" "$HERE/monitor.py" "$HERE/collect-env.sh" "$HERE/pg-stats.sql" "ubuntu@$API_HOST:/tmp/"
runner 'mkdir -p /tmp/loadtest'
scp -q "${SSH_OPTS[@]}" "$HERE/monitor.py" "$HERE/shop.js" "$HERE/prepare.js" "ubuntu@$RUNNER_HOST:/tmp/loadtest/"

if ! runner 'test -s /tmp/loadtest/tokens.json'; then
  echo "logging in the customers and tablets (once)"
  runner "cd /tmp/loadtest && k6 run -q -e BASE_URL=$BASE_URL -e OUT=tokens.json prepare.js"
fi

echo "recording the setup"
api 'cd the-food-app && bash /tmp/collect-env.sh' > "$OUT/env-before.txt"
runner 'echo "k6: $(k6 version)"; echo "vCPUs: $(nproc)"; echo "memory: $(free -m | awk "/Mem:/ {print \$2}") MB"' > "$OUT/runner.txt"
cat > "$OUT/test.txt" <<EOF
name:             $NAME
started (UTC):    $(date -u +%FT%TZ)
target:           $BASE_URL
multipliers:      $MULTS (1x = 17 orders a second)
step length:      $DURATION, with $REST s rest between steps
tablets:          $TABLETS, each asking for new orders every $TABLET_INTERVAL s
EOF

# Remember where nginx's error log ends now, to save only what this test adds.
NGINX_LOG_START=$(api 'sudo stat -c %s /var/log/nginx/error.log')

echo "starting the monitors"
api 'A=$(sudo docker inspect -f "{{.Id}}" the-food-app-prod-api-1); P=$(sudo docker inspect -f "{{.Id}}" the-food-app-prod-postgres-1);
  sudo nohup python3 /tmp/monitor.py --out /tmp/monitor.csv --part nginx=nginx.service \
    --part api=docker-$A.scope --part postgres=docker-$P.scope > /dev/null 2>&1 &'
runner 'nohup python3 /tmp/loadtest/monitor.py --out /tmp/loadtest/monitor.csv --port 0 > /dev/null 2>&1 &'
stop_monitors() {
  api 'sudo pkill -f "^python3 /tmp/monitor.py"' || true
  runner 'pkill -f "^python3 /tmp/loadtest/monitor.py"' || true
  scp -q "${SSH_OPTS[@]}" "ubuntu@$API_HOST:/tmp/monitor.csv" "$OUT/monitor-api.csv" || true
  scp -q "${SSH_OPTS[@]}" "ubuntu@$RUNNER_HOST:/tmp/loadtest/monitor.csv" "$OUT/monitor-runner.csv" || true
  api "sudo tail -c +$((NGINX_LOG_START + 1)) /var/log/nginx/error.log" > "$OUT/nginx-error.log" || true
}
trap stop_monitors EXIT
sleep 10 # a few quiet samples before the first step

echo "multiplier,start_utc,end_utc,nginx_errors" > "$OUT/steps.csv"
for M in $MULTS; do
  echo "=== step ${M}x"
  pg_stats "$OUT/pg-${M}x-before.json"
  LOG_BEFORE=$(api 'sudo wc -l < /var/log/nginx/error.log')
  START=$(date -u +%FT%TZ)
  runner "cd /tmp/loadtest && ulimit -n 65535 && k6 run -q \
    -e BASE_URL=$BASE_URL -e MULT=$M -e DURATION=$DURATION \
    -e TABLETS=$TABLETS -e TABLET_INTERVAL=$TABLET_INTERVAL \
    -e TOKENS=tokens.json -e OUT=k6-${M}x.json shop.js" | tee "$OUT/k6-${M}x.txt" || true
  END=$(date -u +%FT%TZ)
  pg_stats "$OUT/pg-${M}x-after.json"
  scp -q "${SSH_OPTS[@]}" "ubuntu@$RUNNER_HOST:/tmp/loadtest/k6-${M}x.json" "$OUT/"
  NGINX_ERRORS=$(( $(api 'sudo wc -l < /var/log/nginx/error.log') - LOG_BEFORE ))
  echo "$M,$START,$END,$NGINX_ERRORS" >> "$OUT/steps.csv"
  echo "nginx wrote $NGINX_ERRORS lines to its error log in this step"

  # Stop when menus are slower than the requirement, or too many requests fail.
  P99=$(jq '.metrics["http_req_duration{name:menu}"].values["p(99)"]' "$OUT/k6-${M}x.json")
  FAILED=$(jq '.metrics.http_req_failed.values.rate' "$OUT/k6-${M}x.json")
  if jq -en "$P99 > 1000 or $FAILED > 0.01" > /dev/null; then
    echo "stopping: menu p99 ${P99} ms, failed $(jq -n "$FAILED * 100") %"
    break
  fi
  sleep "$REST"
done

api 'cd the-food-app && bash /tmp/collect-env.sh' > "$OUT/env-after.txt"
echo "results in $OUT"
