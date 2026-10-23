-- +goose Up
-- Everyone who logs in: customers, and the people who work at a restaurant.
CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Empty until the app asks for it. No request sets a name yet.
    name          text,
    phone         text NOT NULL UNIQUE,
    role          text NOT NULL CHECK (role IN ('customer', 'restaurant')),
    -- Set only for restaurant staff: the restaurant they work for.
    restaurant_id bigint REFERENCES restaurants (id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CHECK ((role = 'restaurant') = (restaurant_id IS NOT NULL))
);

-- +goose Down
DROP TABLE users;
