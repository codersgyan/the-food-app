# Load the settings from .env, if it exists (cp .env.example .env).
-include .env
export

.PHONY: db db-down migrate migrate-down run seed seed-big test proxy cloud-up cloud-loadtest cloud-down

# Start Postgres and wait until it accepts connections.
db:
	docker compose up -d --wait postgres

# Stop Postgres. Your data stays in the pgdata volume.
db-down:
	docker compose down

# Apply every new migration.
migrate:
	go run ./cmd/migrate up

# Roll back the most recent migration.
migrate-down:
	go run ./cmd/migrate down

# Run the API server on :8080.
run:
	go run ./cmd/api

# Empty the database and fill it with sample restaurants, dishes and users.
seed:
	go run ./cmd/seed

# Run every test. Needs Postgres running (make db).
test:
	go test ./...

# The dataset from the indexes lesson: 20,000 restaurants with 100 dishes
# each, a staff login for every restaurant, and 10,000 customers.
seed-big:
	go run ./cmd/seed -restaurants 20000 -dishes 100 -customers 10000

# Create the two servers on AWS and set them up (see infra/README.md).
cloud-up:
	./infra/up.sh

# Run a load test on them: make cloud-loadtest NAME=customers-only STEPS="1 2 3 5 10"
cloud-loadtest:
	cd loadtest && \
	API_HOST=$$(terraform -chdir=../infra output -raw api_ip) \
	RUNNER_HOST=$$(terraform -chdir=../infra output -raw runner_ip) \
	SSH_KEY=../infra/$$(terraform -chdir=../infra output -raw ssh_key) \
	KNOWN_HOSTS=../infra/secrets/known_hosts \
	BASE_URL=https://$$(terraform -chdir=../infra output -raw domain) \
	./run.sh "$(NAME)" "$(STEPS)"

# Delete everything on AWS, and the DNS record.
cloud-down:
	./infra/down.sh

# Start nginx in front of the API server, on localhost:8000.
proxy:
	docker compose --profile proxy up -d nginx
