.PHONY: db db-down

# Start Postgres and wait until it accepts connections.
db:
	docker compose up -d --wait postgres

# Stop Postgres. Your data stays in the pgdata volume.
db-down:
	docker compose down
