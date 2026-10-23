# Servers on AWS, with one command

This folder creates the two servers our load tests run on, sets them up, and deletes them again. We use it after each module of the course: create the servers, run the tests, keep the results, and delete everything, so nothing keeps costing money.

It uses Terraform, a tool that creates cloud resources from files that describe them. `main.tf` describes everything:

- **The API server** (`m7g.large`: 2 ARM vCPUs, 8 GB): our app and Postgres in Docker, with nginx in front, set up exactly like [deploy/README.md](../deploy/README.md).
- **The test runner** (`c7g.xlarge`: 4 ARM vCPUs), in the same zone, which sends the load with k6. It has more CPU than the API server, so the tool never limits the test.
- A firewall for each: SSH only from your IP address, and only ports 80 and 443 open on the API server.
- A new SSH key, saved in `infra/secrets/`.
- The DNS record for the domain, on Cloudflare.
- A safety net: each server shuts itself down and is deleted after 4 hours, in case you forget to delete them.

## What you need

- **Terraform** 1.6 or newer ([install it](https://developer.hashicorp.com/terraform/install)). OpenTofu works too: use `tofu` where it says `terraform`.
- **An AWS account**, with an access key for a user that can manage EC2 (instances, security groups, key pairs) and read SSM parameters. Terraform reads the key from `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, or from `AWS_PROFILE`.
- **A domain on Cloudflare**, and a Cloudflare API token with two permissions on that zone: Zone → Zone → Read, and Zone → DNS → Edit. Terraform reads it from `CLOUDFLARE_API_TOKEN`.

To use your own domain instead of ours, set these before running anything:

```sh
export TF_VAR_domain=food-api.example.com TF_VAR_cloudflare_zone=example.com
export REPO=https://github.com/<you>/the-food-app.git   # if you test your own fork
```

If the domain already has an `A` record you made by hand, delete it in Cloudflare first. Terraform makes its own, and two records would send some visitors to the wrong server.

## Use it

From the repo's root folder:

```sh
export AWS_ACCESS_KEY_ID=… AWS_SECRET_ACCESS_KEY=… CLOUDFLARE_API_TOKEN=…
export EMAIL=you@example.com   # Let's Encrypt writes here about the certificate

make cloud-up                                              # about 10 minutes
make cloud-loadtest NAME=customers-only STEPS="1 2 3 5 10"
TABLETS=20000 make cloud-loadtest NAME=with-tablets STEPS="1 2 3"
make cloud-down                                            # deletes everything
```

`make cloud-up` runs `infra/up.sh`, which:

1. creates everything in `main.tf` with `terraform apply`
2. waits for both servers to start
3. runs `deploy/setup.sh` on the API server, so you see each step of the deploy guide as it happens, and seeds the dataset from the indexes lesson (`SEED=big`)
4. installs k6 on the test runner

`make cloud-loadtest` runs `loadtest/run.sh` against these servers, and saves the results in `loadtest/results/`. See [loadtest/README.md](../loadtest/README.md) for what a test sends and records.

`make cloud-down` runs `infra/down.sh`, which deletes everything `make cloud-up` created, including the DNS record.

## The certificate

Let's Encrypt gives a domain at most 5 identical certificates a week. So we don't ask for a new one every time: after setting up the server, `up.sh` downloads `/etc/letsencrypt` to `infra/secrets/letsencrypt-<domain>.tar.gz`, and the next `make cloud-up` puts it back before certbot runs. certbot sees a valid certificate and keeps it. When the certificate has less than 30 days left, certbot renews it on the server, and `down.sh` saves the new one.

That file holds the certificate's private key, so it stays in `infra/secrets/`, which git ignores. Never commit it.

## What it costs

AWS bills these servers by the second, and only while they exist. Together they cost well under a dollar an hour, and `make cloud-down` stops the bill. Check the current prices for your region on the [EC2 pricing page](https://aws.amazon.com/ec2/pricing/on-demand/).

## Files

| File | What it is |
| --- | --- |
| `versions.tf` | the Terraform providers we use, for AWS, Cloudflare and a few helpers |
| `variables.tf` | everything you can change, like the domain, region and machine sizes |
| `main.tf` | the servers, firewalls, SSH key and DNS record |
| `outputs.tf` | what `make cloud-loadtest` needs to know: the IP addresses and the SSH key |
| `up.sh`, `down.sh` | create and set up, and delete |
| `.terraform.lock.hcl` | the exact provider versions, so everyone gets the same ones |
