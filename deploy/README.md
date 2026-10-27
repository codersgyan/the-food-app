# Running it on a server

These are the steps to run the app on one Linux server, behind nginx, at `https://food-api.sys0.in`. It's the design from the reverse proxy lesson: nginx answers HTTPS on port 443 and passes each request to our API server on port 8080 of the same machine.

Our API server and Postgres run in Docker, with `compose.prod.yaml`. nginx runs on the server itself, installed with apt, because then certbot can renew its certificate and reload it without any extra setup. The nginx in `compose.yaml` (`make proxy`) is only for trying the proxy on your laptop.

We tested these steps on Ubuntu 26.04 LTS, on an AWS `m7g.large` (2 ARM vCPUs, 8 GB of memory) in Mumbai. They work on Ubuntu 24.04 too. Give the server a disk of at least 20 GB: the first build takes about 3 GB for Docker's images and its build cache, which fills three quarters of the default 8 GB disk on AWS.

`deploy/setup.sh` runs steps 2 to 5 for you, in the same order, once step 1 is done: `DOMAIN=food-api.sys0.in EMAIL=you@example.com ./deploy/setup.sh`. The `infra/` folder goes one step further, and creates the servers too. The steps below are what both of them do, one command at a time.

## 1. Point the domain at the server, and open its ports

Add a DNS `A` record for `food-api.sys0.in` with the server's IP address. Check it before going on:

```sh
dig +short food-api.sys0.in
```

Cloud servers have a firewall outside the machine too. On AWS, it's the instance's security group. Open two ports in it, to everyone: 80 (HTTP), so Let's Encrypt can check the domain and nginx can send visitors to HTTPS, and 443 (HTTPS), for the app. Keep 8080 closed: only nginx, on the same machine, talks to our API server.

## 2. Install Docker and get the code

Install Docker from Docker's own apt repository. It comes with Compose, and it's newer than Ubuntu's `docker.io` package, which doesn't include Compose. These are the steps from Docker's install guide for Ubuntu (https://docs.docker.com/engine/install/ubuntu/):

```sh
sudo apt update
sudo apt install ca-certificates curl
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
sudo apt update
sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo docker compose version
```

Then clone the repo. Every command from now on runs inside this folder on the server:

```sh
git clone https://github.com/codersgyan/the-food-app.git
cd the-food-app
```

## 3. Make the settings file

The settings live in `/etc/food-api.env`, outside the repo, so they never end up in git. On a server, it needs only three lines: the two secrets, and a password for Postgres. Never reuse the values from `.env.example`, because they're in a public repo. This makes new random ones:

```sh
sudo tee /etc/food-api.env > /dev/null <<EOF
JWT_SECRET=$(openssl rand -hex 32)
PAYMENT_WEBHOOK_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
EOF
sudo chmod 600 /etc/food-api.env
```

`chmod 600` makes the file readable only by root, because it holds the secrets. You can add any other setting from `.env.example` to it, like `FAKEPAY_DELAY`. Don't set `ADDR` or `DATABASE_URL`: `compose.prod.yaml` sets them for the containers.

## 4. Start the API server and Postgres

```sh
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env up -d --build
```

This builds our image from the `Dockerfile`, and starts Postgres and then the API server. Both restart by themselves if they stop, and when the server reboots.

The build leaves about 1.5 GB of cache behind. On a small disk, free it:

```sh
sudo docker builder prune -f
```

Then create the tables, and add the sample data. `run --rm` starts a one-off container from the same image, runs the program, and removes the container:

```sh
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env run --rm api migrate up
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env run --rm api seed
curl -s 127.0.0.1:8080/v1/restaurants/1/menu
```

The API server is published only on `127.0.0.1:8080`, so nginx on this machine can reach it, and the internet can't. This is important, because ports that Docker publishes on every address skip the ufw firewall. So `"8080:8080"` would open our API server to the whole internet, even with ufw on. Postgres isn't published at all, so only the API container can reach it.

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

The first time, certbot asks for an email address, where Let's Encrypt warns you if a certificate is about to expire, and asks you to agree to the Let's Encrypt Subscriber Agreement. It saves the certificate in `/etc/letsencrypt/live/food-api.sys0.in/`, which is where our nginx config looks for it.

Ubuntu's nginx accepts at most 768 connections per worker process. Each request through nginx uses two: one from the client to nginx, and one from nginx to our API server. nginx runs one worker per CPU, so on a 2-CPU server that's only about 768 clients at once, and after that nginx refuses new connections. So allow 4,096 connections per worker, and let each worker open enough files for them, because every connection is an open file:

```sh
sudo sed -i 's/worker_connections 768;/worker_connections 4096;/' /etc/nginx/nginx.conf
sudo sed -i '/^worker_processes/a worker_rlimit_nofile 8192;' /etc/nginx/nginx.conf
```

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

## Updating to a new version

On the server, get the new code, rebuild the image, and apply any new migrations:

```sh
git pull
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env up -d --build
sudo docker compose -f compose.prod.yaml --env-file /etc/food-api.env run --rm api migrate up
```

The data stays in the `pgdata` volume, so rebuilding doesn't touch it. Don't run `seed` again, because it empties every table.
