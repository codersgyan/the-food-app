-- +goose Up
-- Showing an order means finding its dishes by order_id.
CREATE INDEX order_items_order_id_idx ON order_items (order_id);

-- +goose Down
DROP INDEX order_items_order_id_idx;
