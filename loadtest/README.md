# Load test

This folder tests the claims the path makes about our app, on real servers: that one server and one database handle the dinner rush, that the index keeps menus fast, and where the design stops being enough.

It runs on two machines in the same AWS zone: the API server, set up with [deploy/README.md](../deploy/README.md), and a separate test runner that sends the traffic. The test runner is a different machine, so the tool that sends the load never competes with the server for CPU.

## What a test sends

The traffic follows the numbers in the lessons. At the dinner peak, the app gets 120,000 orders in two hours, which is about 17 orders a second. That's **1x**. `MULT=2` doubles the customers, and so on.

**Customers.** A new customer visit starts at a fixed rate, whether or not the server keeps up, the way real customers keep arriving. In each visit:

1. 1 in 10 visits starts with a login: `POST /v1/login`, then `POST /v1/login/verify`. The others are already logged in.
2. `GET /v1/me`.
3. `GET /v1/restaurants?near=…`, from a random spot in Bengaluru: the first page, the nearest 20 restaurants.
4. Four menus from the nearest restaurants, `GET /v1/restaurants/{id}/menu`.
5. 4 in 10 visits then order: `POST /v1/orders` with one to three dishes, `POST /v1/orders/{id}/pay`, and then `GET /v1/orders/{id}` every second until the payment goes through. The other visits only browse.

Between steps, the customer pauses for half a second to a second and a half, like a person reading the screen. That makes about 20 requests for every order, the number the lessons use, and about 340 requests a second at 1x.

**Restaurant tablets.** With `TABLETS=20000`, the tablets of the first 20,000 restaurants take turns asking `GET /v1/restaurants/{id}/orders?status=paid`, so each one asks once every `TABLET_INTERVAL` seconds. When it finds an order, it accepts it and marks it ready. The tablets don't grow with `MULT`: a tablet asks every few seconds however busy the restaurant is.

Requests ask for gzip, like a phone does, so nginx compresses the responses.

## What a test records

`run.sh` writes everything into `results/<date>-<name>/`:

| File | What's in it |
| --- | --- |
| `test.txt` | the test's settings |
| `env-before.txt`, `env-after.txt` | the API server: machine, versions, Postgres and nginx settings, row counts, nginx config, git commit |
| `runner.txt` | the test runner's machine and k6 version |
| `steps.csv` | when each step started and ended, in UTC, and how many lines nginx wrote to its error log |
| `nginx-error.log` | everything nginx wrote to its error log during the test |
| `k6-<n>x.txt` | each step's results: count, rate, p50, p95, p99, max and failures for every kind of request |
| `k6-<n>x.json` | everything k6 measured in that step |
| `pg-<n>x-before.json`, `pg-<n>x-after.json` | Postgres's counters around each step: transactions, pages read from memory and from disk, and sequential and index scans for each table |
| `monitor-api.csv` | every 5 seconds on the API server: CPU in cores for the machine, nginx, the API server and Postgres, their memory, network traffic, and open HTTPS connections |
| `monitor-runner.csv` | the same for the test runner, to show it wasn't the bottleneck |

The `login` and `verify` times include the fake OTP service's delay of 200 to 500 ms, on purpose. So read the row for each kind of request, not only the `ALL` row.

## Run it

You need the API server from the deploy guide, seeded with the dataset from the indexes lesson, k6 on the test runner, and jq on your laptop.

On the API server, seed it:

```sh
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env run --rm api seed -restaurants 20000 -dishes 100 -customers 10000
```

Then, from your laptop:

```sh
export API_HOST=<api server IP> RUNNER_HOST=<test runner IP> SSH_KEY=<your .pem file>
cd loadtest

# Customers only, at 1x, 2x, 3x, 5x and 10x.
./run.sh customers-only "1 2 3 5 10"

# The same, with every restaurant's tablet asking every 10 seconds.
TABLETS=20000 TABLET_INTERVAL=10 ./run.sh with-tablets "1 2 3 5 10"
```

Each step lasts 3 minutes (`DURATION`), with 30 seconds of rest between steps (`REST`), and starts with the orders tables empty, like the start of a dinner rush. A step passes when, for the whole step, menus stay under 1 second at p99 (the requirement from the lessons), under 1% of requests fail, and k6 drops under 1% of the visits because the server couldn't take them in time. The test stops at the first step that doesn't pass.

To find the limit more exactly, give it one multiplier that passed and one that failed, with `SEARCH=1`. It tries the middle, keeps the half the limit is in, and stops when the two are 0.1 apart. It writes the answer to `limit.txt`:

```sh
SEARCH=1 ./run.sh customers-only-limit "1 2"
```

`SEARCH=tablets` does the same with the number of tablets, keeping the customers steady at `MULT`, and stops when the two are 1,000 tablets apart:

```sh
SEARCH=tablets MULT=1 ./run.sh tablets-limit "0 20000"
```

The first run logs in 2,000 customers and all 20,000 tablets once, with `prepare.js`, and keeps their tokens on the test runner.
