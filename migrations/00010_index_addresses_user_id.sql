-- +goose Up
-- GET /v1/me finds a customer's addresses by user_id. Without this index,
-- Postgres reads every address to find them. This was missing at first: the
-- load test with 10,000 customers showed it, as 76 million address rows read
-- in three minutes at the dinner peak.
CREATE INDEX addresses_user_id_idx ON addresses (user_id);

-- +goose Down
DROP INDEX addresses_user_id_idx;
