-- +goose Up
CREATE TABLE dishes (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    restaurant_id bigint NOT NULL REFERENCES restaurants (id),
    name          text NOT NULL,
    -- In paise, so ₹250 is 25000. Whole numbers keep money exact.
    price         integer NOT NULL CHECK (price > 0),
    sold_out      boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE dishes;
