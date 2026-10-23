-- +goose Up
-- GET /v1/restaurants/{id}/menu finds a restaurant's dishes by restaurant_id.
-- Without this index, Postgres reads every dish to find them.
CREATE INDEX dishes_restaurant_id_idx ON dishes (restaurant_id);

-- +goose Down
DROP INDEX dishes_restaurant_id_idx;
