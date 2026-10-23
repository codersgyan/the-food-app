.PHONY: db db-down migrate migrate-down

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
