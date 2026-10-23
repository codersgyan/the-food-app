# Running it on a server

These are the steps to run the app on one Linux server, behind nginx, at `https://food-api.sys0.in`. It's the design from the reverse proxy lesson: nginx answers HTTPS on port 443 and passes each request to our API server on port 8080 of the same machine.

The commands are for Ubuntu 24.04. You need a server with a public IP address, and Postgres running somewhere the server can reach.

## 1. Point the domain at the server

Add a DNS `A` record for `food-api.sys0.in` with the server's IP address. Check it before going on:

```sh
dig +short food-api.sys0.in
```

## 2. Build and install the API server

On your laptop, build a Linux binary and copy it to the server:

```sh
GOOS=linux GOARCH=amd64 go build -o food-api ./cmd/api
scp food-api you@food-api.sys0.in:/tmp/
```

On the server:

```sh
sudo install /tmp/food-api /usr/local/bin/food-api
```

## 3. Give it its settings

Create `/etc/food-api.env`. `ADDR=127.0.0.1:8080` makes the server listen only inside the machine, so every request has to go through nginx. Use your own long random secrets, never the defaults:

```sh
ADDR=127.0.0.1:8080
DATABASE_URL=postgres://food:…@…:5432/food
JWT_SECRET=…
PAYMENT_WEBHOOK_SECRET=…
FAKEPAY_WEBHOOK_URL=http://127.0.0.1:8080/payments/webhook
```

Create the tables from your laptop, pointing at the server's database:

```sh
DATABASE_URL=postgres://food:…@…:5432/food make migrate seed
```

## 4. Start the API server

```sh
sudo cp deploy/food-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now food-api
curl -s 127.0.0.1:8080/v1/restaurants/1/menu
```

## 5. Get a certificate, then start nginx

Install nginx and certbot. certbot gets a free HTTPS certificate from Let's Encrypt, and renews it before it expires:

```sh
sudo apt install nginx certbot python3-certbot-nginx
sudo certbot certonly --nginx -d food-api.sys0.in
```

Then install our config and reload nginx:

```sh
sudo cp deploy/nginx/food-api.sys0.in.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/food-api.sys0.in.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 6. Check it from your laptop

```sh
curl https://food-api.sys0.in/v1/restaurants/1/menu
curl -sI -H 'Accept-Encoding: gzip' https://food-api.sys0.in/v1/restaurants/1/menu | grep -i content-encoding
curl -s -o /dev/null -w '%{http_code}\n' http://food-api.sys0.in:8080/v1/restaurants/1/menu
```

The first one returns the menu, and the second one shows that nginx compressed it. The last one has to fail: port 8080 is closed to the internet.
