-- +goose Up
-- GET /v1/me finds a customer's addresses by user_id. Without this index,
-- Postgres reads every address to find them.
CREATE INDEX addresses_user_id_idx ON addresses (user_id);

-- +goose Down
DROP INDEX addresses_user_id_idx;
