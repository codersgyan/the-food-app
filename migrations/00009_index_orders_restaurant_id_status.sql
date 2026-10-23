-- +goose Up
-- A restaurant asks for its own orders in one status, like the paid ones
-- waiting for the kitchen. restaurant_id comes first, because a restaurant
-- always asks for its own orders.
CREATE INDEX orders_restaurant_id_status_idx ON orders (restaurant_id, status);

-- +goose Down
DROP INDEX orders_restaurant_id_status_idx;
