# Running it on a server

These are the steps to run the app on one Linux server, behind nginx, at `https://food-api.sys0.in`. It's the design from the reverse proxy lesson: nginx answers HTTPS on port 443 and passes each request to our API server on port 8080 of the same machine.

The commands are for Ubuntu 24.04. You need a server with a public IP address, and Postgres running somewhere the server can reach.

On the server, nginx is installed with apt and runs on its own, not in Docker. The nginx in `compose.yaml` (`make proxy`) is only for trying the proxy on your laptop, with no domain and no certificate.

## 1. Point the domain at the server

Add a DNS `A` record for `food-api.sys0.in` with the server's IP address. Check it before going on:

```sh
dig +short food-api.sys0.in
```

## 2. Get the code and the API server onto the server

On the server, clone the repo. The config files in `deploy/` come from here, and every command from now on runs inside this folder on the server, unless it says "on your laptop":

```sh
git clone https://github.com/codersgyan/the-food-app.git
cd the-food-app
```

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

The server reads its settings from `/etc/food-api.env`. It's the same `KEY=value` format as `.env` on your laptop, and systemd loads it when it starts the server. We start from `.env.example`.

Make two new secrets. Never reuse the values from `.env.example`: they're in a public repo.

```sh
openssl rand -hex 32   # for JWT_SECRET
openssl rand -hex 32   # for PAYMENT_WEBHOOK_SECRET
```

Then install the file, and make it readable only by root, because it holds the secrets:

```sh
sudo install -m 600 .env.example /etc/food-api.env
sudo nano /etc/food-api.env
```

Put the two new secrets in, and set these lines too. `ADDR=127.0.0.1:8080` makes the server listen only inside the machine, so every request has to go through nginx:

```sh
JWT_SECRET=<the first secret>
PAYMENT_WEBHOOK_SECRET=<the second secret>
ADDR=127.0.0.1:8080
DATABASE_URL=postgres://food:…@…:5432/food
FAKEPAY_WEBHOOK_URL=http://127.0.0.1:8080/payments/webhook
```

On your laptop, create the tables, pointing at the server's database:

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

HTTPS needs a certificate for `food-api.sys0.in`. We get a free one from Let's Encrypt, using a program called certbot. Before Let's Encrypt gives us one, it checks that we really control the domain: certbot puts a small file on our server, and Let's Encrypt fetches it over plain HTTP. So port 80 has to be open. If the server runs the ufw firewall, open ports 80 and 443:

```sh
sudo ufw allow 'Nginx Full'
```

Install nginx and certbot, and get the certificate:

```sh
sudo apt install nginx certbot python3-certbot-nginx
sudo certbot certonly --nginx -d food-api.sys0.in \
  --deploy-hook "systemctl reload nginx"
```

certbot saves the certificate in `/etc/letsencrypt/live/food-api.sys0.in/`, which is where our nginx config looks for it.

Then install our config and reload nginx:

```sh
sudo cp deploy/nginx/food-api.sys0.in.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/food-api.sys0.in.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### How the certificate renews itself

A Let's Encrypt certificate lasts 90 days. The certbot package installs a timer that runs `certbot renew` twice a day, and it renews any certificate that has less than 30 days left. You can see the timer here:

```sh
systemctl list-timers certbot.timer
```

Renewing writes a new certificate to the disk, but nginx keeps the old one in memory until it's reloaded. That's what `--deploy-hook` is for: after every renewal, certbot runs `systemctl reload nginx`, so nginx picks up the new certificate. certbot saves the hook in `/etc/letsencrypt/renewal/food-api.sys0.in.conf`, so you give it only once. Without it, every renewal would work, and the site would still stop working on the day the old certificate expires.

To check that renewal works, without waiting two months:

```sh
sudo certbot renew --dry-run
```

It goes through the whole renewal against Let's Encrypt's test servers, and changes nothing.

## 6. Check it from your laptop

```sh
curl https://food-api.sys0.in/v1/restaurants/1/menu
curl -sI -H 'Accept-Encoding: gzip' https://food-api.sys0.in/v1/restaurants/1/menu | grep -i content-encoding
curl -s -o /dev/null -w '%{http_code}\n' http://food-api.sys0.in:8080/v1/restaurants/1/menu
```

The first one returns the menu, and the second one shows that nginx compressed it. The last one has to fail: port 8080 is closed to the internet.
