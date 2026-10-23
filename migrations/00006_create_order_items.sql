-- +goose Up
-- One row for each dish in each order.
--
-- No primary key, as in the course's data model. A primary key on
-- (order_id, dish_id) would also index order_id, and the indexes lesson adds
-- that index on its own. The server makes sure a dish appears only once in
-- an order.
CREATE TABLE order_items (
    order_id bigint NOT NULL REFERENCES orders (id),
    dish_id  bigint NOT NULL REFERENCES dishes (id),
    quantity integer NOT NULL CHECK (quantity > 0),
    -- The dish's price when the order was placed, in paise. Copied from
    -- dishes, so the bill never changes when the menu does.
    price    integer NOT NULL CHECK (price > 0)
);

-- +goose Down
DROP TABLE order_items;
