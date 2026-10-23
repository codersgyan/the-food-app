# The food app

This is the food delivery app we design in the system design course, written in Go. It has every request we designed for the customer's app and the restaurant's tablet, and the six tables from the data modeling lesson.

We'll keep coming back to this app as the course goes on. When a lesson says something about it, like "one server is enough at the dinner rush", we test that claim on this code.

## What you need

- Go 1.27 or newer
- Docker, to run Postgres

## Run it

```sh
cp .env.example .env   # your settings, including two secrets
make db        # start Postgres 18 in Docker
make migrate   # create the tables and indexes
make seed      # add 20 restaurants in Bengaluru, their dishes, and some users
make run       # start the server on localhost:8080
```

Postgres listens on port 5440 on your machine, so it doesn't clash with a Postgres you may already run on 5432.

`make seed` empties every table first, so you can run it again whenever you want to start over.

## Try it

Run these in another terminal while the server is running. They follow the same journey as the lessons. You need `jq` for the lines that save the token, or you can copy the token by hand.

First, the customer logs in with their phone number. The server asks the OTP service to text a code to the phone:

```sh
curl -X POST localhost:8080/v1/login -d '{"phone": "+919800000001"}'
```

The OTP service here is a fake, so no text message is sent, and the code is always `123456`. The customer sends it back and gets a token:

```sh
TOKEN=$(curl -s -X POST localhost:8080/v1/login/verify \
  -d '{"phone": "+919800000001", "code": "123456"}' | jq -r .token)
```

Every customer request after this sends the token. Let's see who we are, and our saved addresses:

```sh
curl localhost:8080/v1/me -H "Authorization: Bearer $TOKEN"
```

Add another address:

```sh
curl -X POST localhost:8080/v1/addresses -H "Authorization: Bearer $TOKEN" \
  -d '{"address": "Flat 5, 1st Cross, Koramangala"}'
```

Find restaurants near the phone. The phone sends its own spot on the map, and the server returns the restaurants within 7 km, nearest first:

```sh
curl "localhost:8080/v1/restaurants?near=12.9352,77.6245"
```

Open restaurant 1's menu. Prices are in paise, so `25000` is ₹250:

```sh
curl localhost:8080/v1/restaurants/1/menu
```

Place an order for two chicken biryanis and a raita, to be delivered to address 1. Notice that it doesn't send any prices. The server reads them from the menu.

```sh
curl -X POST localhost:8080/v1/orders -H "Authorization: Bearer $TOKEN" \
  -d '{"restaurant_id": 1, "address_id": 1, "items": [{"dish_id": 1, "quantity": 2}, {"dish_id": 4, "quantity": 1}]}'
```

Pay for it. The server answers at once, with the order in `paying`:

```sh
curl -X POST localhost:8080/v1/orders/1/pay -H "Authorization: Bearer $TOKEN"
```

Run the same command again, as if the customer tapped **Pay** twice. The server answers 409, because the order isn't waiting for payment any more, and it doesn't charge again.

Now check the order. For a second or two it says `paying`. Then the fake payment provider calls our webhook, as if the customer typed their UPI PIN, and the order says `paid`:

```sh
curl localhost:8080/v1/orders/1 -H "Authorization: Bearer $TOKEN"
```

Now the restaurant's tablet. Restaurant 1's staff log in with `+919810000001`, restaurant 2's with `+919810000002`, and so on:

```sh
STAFF=$(curl -s -X POST localhost:8080/v1/login/verify \
  -d '{"phone": "+919810000001", "code": "123456"}' | jq -r .token)
```

The tablet asks for the orders it has received and still needs to cook:

```sh
curl "localhost:8080/v1/restaurants/1/orders?status=paid" -H "Authorization: Bearer $STAFF"
```

The kitchen accepts the order, and later marks it ready:

```sh
curl -X POST localhost:8080/v1/orders/1/accept -H "Authorization: Bearer $STAFF"
curl -X POST localhost:8080/v1/orders/1/ready -H "Authorization: Bearer $STAFF"
```

And the raita has run out, so the tablet marks it sold out:

```sh
curl -X PATCH localhost:8080/v1/dishes/4 -H "Authorization: Bearer $STAFF" -d '{"sold_out": true}'
```

## Try it in Bruno

If you prefer clicking to typing curl commands, the `bruno` folder is a collection for [Bruno](https://www.usebruno.com), an API client that keeps its collections as plain files in your repo. Open it in Bruno (version 3 or newer). The values every request uses, like the server's address and the seeded phone numbers, are collection variables, so there's no environment to pick. You'll find them in the collection's settings, under Variables.

It has a folder for each kind of client: the customer's app, the restaurant's tablet, and the payment provider. The requests are in the same order as the journey above. Run "Verify login code" first: it saves the customer's token, and the other customer requests send it. "Staff login" does the same for the tablet. The tokens live only in Bruno's memory, so they never end up in git.

The payment provider's webhook is there too, signed with the same secret the server checks. Change the `webhookSecret` variable, and the server answers 401.

To run the whole collection from the terminal, while the server is running:

```sh
cd bruno
npx @usebruno/cli run -r
```

Run the server with `FAKEPAY_DELAY=0s make run` for this, so each order is paid before the kitchen tries to accept it.

## Put nginx in front of it

In the reverse proxy lesson, we put nginx in front of our API server. You can run the same thing on your laptop:

```sh
make proxy
```

nginx listens on localhost:8000 and passes requests to the server from `make run`. Ask it for a menu with compression on, and you'll see the `Content-Encoding: gzip` header that nginx added:

```sh
curl -sI -H 'Accept-Encoding: gzip' localhost:8000/v1/restaurants/1/menu
```

The config is in `deploy/nginx/local.conf`. The one for the real server, with HTTPS, is `deploy/nginx/food-api.sys0.in.conf`, and [deploy/README.md](deploy/README.md) has the steps to run the app at `https://food-api.sys0.in`.

## Run it on a server

[deploy/README.md](deploy/README.md) has every step to run the app on one Ubuntu server at `https://food-api.sys0.in`: the API server and Postgres in Docker, with nginx in front of them on the server itself, and a free HTTPS certificate from Let's Encrypt that renews itself.

## Every request

Here's every request, as we designed them in the API lesson.

| Request | Who sends it | What it does |
| --- | --- | --- |
| `POST /v1/login` | customer or staff | texts a login code to the phone |
| `POST /v1/login/verify` | customer or staff | checks the code, returns a token |
| `GET /v1/me` | customer | the customer, with their saved addresses |
| `POST /v1/addresses` | customer | saves a delivery address |
| `GET /v1/restaurants?near=lat,lng` | customer | restaurants within 7 km, nearest first |
| `GET /v1/restaurants/{id}/menu` | customer | the restaurant with its dishes |
| `POST /v1/orders` | customer | places an order |
| `POST /v1/orders/{id}/pay` | customer | starts the payment |
| `GET /v1/orders/{id}` | customer | the order, with its status |
| `GET /v1/restaurants/{id}/orders?status=paid` | tablet | the orders the kitchen still needs to cook |
| `POST /v1/orders/{id}/accept` | tablet | the kitchen will cook this order |
| `POST /v1/orders/{id}/ready` | tablet | the order is cooked and packed |
| `PATCH /v1/dishes/{id}` | tablet | changes a dish's price, or marks it sold out |
| `POST /payments/webhook` | payment provider | says whether a payment went through |

## The fake OTP service and payment provider

The app talks to two outside services: an OTP service that texts login codes, and a payment provider like Razorpay. This repo has a fake of each, so you can run everything on your laptop, with no accounts and no API keys.

- The fake OTP service accepts `123456` for every phone number, and writes the code to the server's log instead of texting it.
- The fake payment provider approves every payment 1 to 2 seconds after the server asks for it, and then calls `POST /payments/webhook`, signed the way Razorpay signs its webhooks.

Real payment providers aren't always that tidy. Some payments fail, and the same webhook can arrive twice. You can make the fake do both:

```sh
FAKEPAY_FAIL_RATE=0.2 FAKEPAY_DUPLICATE_RATE=0.5 make run
```

Now about 1 in 5 payments fails, and the order goes back to `pending` so the customer can pay again. About half the webhooks arrive twice, and you'll see `payment webhook changed nothing` in the log for the second one.

## Settings

The server reads its settings from environment variables. On your laptop, they come from `.env`, which the Makefile loads, and `.env.example` explains every one. `.env` is in `.gitignore`, so your settings never end up in git.

The two secrets have no default, and the server won't start without them. The code is public, so a default secret would be one that everyone knows: anyone could sign their own login token, or send a fake "payment captured" webhook. `.env.example` has values that are fine on your laptop, and on a server you make your own (see [deploy/README.md](deploy/README.md)).

| Variable | Default | What it is |
| --- | --- | --- |
| `JWT_SECRET` | none, required | signs login tokens |
| `PAYMENT_WEBHOOK_SECRET` | none, required | the secret the payment provider signs webhooks with |
| `ADDR` | `:8080` | where the server listens |
| `DATABASE_URL` | `postgres://food:food@localhost:5440/food?sslmode=disable` | the Postgres to use |
| `NEAR_RADIUS_KM` | `7` | how far a restaurant can be and still deliver |
| `FAKEOTP_DELAY` | `200ms-500ms` | how long the fake OTP service takes to answer |
| `FAKEPAY_DELAY` | `1s-2s` | how long the customer takes to approve a payment |
| `FAKEPAY_WEBHOOK_URL` | `http://localhost:8080/payments/webhook` | where the fake provider sends webhooks |
| `FAKEPAY_FAIL_RATE` | `0` | the share of payments that fail, from 0 to 1 |
| `FAKEPAY_DUPLICATE_RATE` | `0` | the share of webhooks sent twice, from 0 to 1 |

## Run the tests

```sh
make db     # if Postgres isn't running yet
make test
```

The tests run against a real Postgres, in a separate database called `food_test`, so they never touch your seeded data. They send real HTTP requests to the server, so each test goes through the handler, the service and the SQL. They check the rules from the lessons, like "tapping Pay ten times at once charges the customer once", and "a new price doesn't change an order already placed".

## Load test data

The indexes lesson talks about 20,000 restaurants with 100 dishes each, 2 million dishes in all. This makes exactly that:

```sh
make seed-big
```

It takes about 15 seconds. The first 20 restaurants are the same as in `make seed`, and the rest have made-up names and random spots in Bengaluru. Every restaurant also gets a staff login, so each one's tablet can log in, and there are 10,000 more customers, each with an address. `cmd/seed/main.go` lists their phone numbers.

## How the code is laid out

```
cmd/api/            the server: reads settings, connects to Postgres, starts listening
cmd/migrate/        creates the tables (make migrate)
cmd/seed/           fills the tables with sample data (make seed)
migrations/         one SQL file for each table and each index
internal/handler/   HTTP: reads the request, calls a service, writes the response
internal/service/   the rules: who can do what, and when
internal/store/     the SQL, one file for each table
internal/auth/      login tokens
internal/fake/      the fake OTP service and payment provider
bruno/              the Bruno collection
deploy/             nginx configs, and the steps to run it on a server
Dockerfile          the image for a server: the API server, migrate and seed
compose.prod.yaml   the API server and Postgres on a server
```

A request goes from the handler to a service to the store, and never the other way. Handlers know HTTP but no SQL. Services know the rules but not HTTP. The store knows SQL but no rules. So if you want to find the guard that stops a double charge, the SQL is `SetStatus` in `internal/store/orders.go` (`WHERE id = $1 AND status = $2`), and the rule that moves an order from pending to paying is `Pay` in `internal/service/payments.go`.
