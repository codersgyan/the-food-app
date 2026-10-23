#!/usr/bin/env bash
# run.sh runs one load test from your laptop, on two servers: the API server,
# and a separate test runner that sends the traffic. Everything it measures
# goes into results/<date>-<name>/.
#
#     export API_HOST=3.111.144.254 RUNNER_HOST=3.108.191.183 SSH_KEY=~/.ssh/pg-load-test.pem
#     ./run.sh customers-only "1 2 3 5 10"
#     TABLETS=20000 TABLET_INTERVAL=10 ./run.sh with-tablets "1 2 3 5 10"
#
# It runs one step for each multiplier, and stops at the first step the server
# doesn't keep up with. A step passes when, for the whole step, menus stay
# under 1 second at p99 (the requirement from the lessons), under 1% of
# requests fail, and k6 drops under 1% of the visits because the server
# couldn't take them in time.
#
# To find the limit more exactly, give it a multiplier that passed and one
# that failed, with SEARCH=1. It tries the middle, keeps the half the limit is
# in, and stops when the two are 0.1 apart:
#
#     SEARCH=1 ./run.sh customers-only-limit "1 2"
#
# SEARCH=tablets does the same with the number of tablets, at MULT customers,
# and stops when the two are 1,000 tablets apart:
#
#     SEARCH=tablets MULT=1 ./run.sh tablets-limit "0 20000"
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

echo "multiplier,tablets,start_utc,end_utc,menu_p99_ms,failed_pct,dropped_pct,nginx_errors,passed" > "$OUT/steps.csv"

# run_step runs one step with multiplier $1 and $2 tablets, and sets PASSED to 1 or 0.
run_step() {
  local M=$1 T=$2 S
  S="${M}x-${T}t" # names this step's files
  echo "=== step ${M}x, $T tablets"
  pg_stats "$OUT/pg-$S-before.json"
  local LOG_BEFORE START END NGINX_ERRORS P99 FAILED DROPPED
  LOG_BEFORE=$(api 'sudo wc -l < /var/log/nginx/error.log')
  START=$(date -u +%FT%TZ)
  runner "cd /tmp/loadtest && ulimit -n 65535 && k6 run -q \
    -e BASE_URL=$BASE_URL -e MULT=$M -e DURATION=$DURATION \
    -e TABLETS=$T -e TABLET_INTERVAL=$TABLET_INTERVAL \
    -e TOKENS=tokens.json -e OUT=k6-$S.json shop.js" | tee "$OUT/k6-$S.txt" || true
  END=$(date -u +%FT%TZ)
  pg_stats "$OUT/pg-$S-after.json"
  scp -q "${SSH_OPTS[@]}" "ubuntu@$RUNNER_HOST:/tmp/loadtest/k6-$S.json" "$OUT/"
  NGINX_ERRORS=$(( $(api 'sudo wc -l < /var/log/nginx/error.log') - LOG_BEFORE ))

  P99=$(jq '.metrics["http_req_duration{name:menu}"].values["p(99)"] // 0 | round' "$OUT/k6-$S.json")
  FAILED=$(jq '.metrics.http_req_failed.values.rate * 100 | . * 100 | round / 100' "$OUT/k6-$S.json")
  DROPPED=$(jq '(.metrics.dropped_iterations.values.count // 0) as $d
    | (.metrics.iterations.values.count // 0) as $i
    | if $d + $i == 0 then 0 else ($d / ($d + $i) * 100 | . * 100 | round / 100) end' "$OUT/k6-$S.json")
  if jq -en "$P99 < 1000 and $FAILED < 1 and $DROPPED < 1" > /dev/null; then PASSED=1; else PASSED=0; fi
  echo "$M,$T,$START,$END,$P99,$FAILED,$DROPPED,$NGINX_ERRORS,$PASSED" >> "$OUT/steps.csv"
  echo "menu p99 ${P99} ms, failed ${FAILED}%, dropped ${DROPPED}%, nginx errors $NGINX_ERRORS: $([ "$PASSED" = 1 ] && echo passed || echo FAILED)"
}

if [ "${SEARCH:-0}" = 1 ]; then
  read -r LOW HIGH <<< "$MULTS"
  while jq -en "$HIGH - $LOW > 0.1" > /dev/null; do
    MID=$(jq -n "($LOW + $HIGH) / 2 * 100 | round / 100")
    run_step "$MID" "$TABLETS"
    if [ "$PASSED" = 1 ]; then LOW=$MID; else HIGH=$MID; fi
    sleep "$REST"
  done
  echo "with $TABLETS tablets, the limit is between ${LOW}x (passed) and ${HIGH}x (failed)" | tee "$OUT/limit.txt"
elif [ "${SEARCH:-0}" = tablets ]; then
  M=${MULT:?set MULT, the customers to keep steady while the tablets change}
  read -r LOW HIGH <<< "$MULTS"
  while [ $((HIGH - LOW)) -gt 1000 ]; do
    MID=$(( (LOW + HIGH) / 2 / 100 * 100 ))
    run_step "$M" "$MID"
    if [ "$PASSED" = 1 ]; then LOW=$MID; else HIGH=$MID; fi
    sleep "$REST"
  done
  echo "at ${M}x, with a poll every $TABLET_INTERVAL s, the limit is between $LOW tablets (passed) and $HIGH (failed)" | tee "$OUT/limit.txt"
else
  for M in $MULTS; do
    run_step "$M" "$TABLETS"
    [ "$PASSED" = 1 ] || { echo "stopping: the server didn't keep up at ${M}x"; break; }
    sleep "$REST"
  done
fi

api 'cd the-food-app && bash /tmp/collect-env.sh' > "$OUT/env-after.txt"
echo "results in $OUT"
