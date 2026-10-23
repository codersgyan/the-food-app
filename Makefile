.PHONY: db db-down migrate migrate-down run seed seed-big test proxy

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

# The dataset from the indexes lesson: 20,000 restaurants, 100 dishes each.
seed-big:
	go run ./cmd/seed -restaurants 20000 -dishes 100

# Start nginx in front of the API server, on localhost:8000.
proxy:
	docker compose --profile proxy up -d nginx
