-- +goose Up
CREATE TABLE orders (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id          bigint NOT NULL REFERENCES users (id),
    restaurant_id    bigint NOT NULL REFERENCES restaurants (id),
    placed_at        timestamptz NOT NULL DEFAULT now(),
    -- A copy of the address text, so the order keeps it even if the
    -- customer edits or deletes the saved address later.
    delivery_address text NOT NULL,
    -- In paise.
    total            integer NOT NULL CHECK (total > 0),
    status           text NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'paying', 'paid', 'accepted', 'ready')),
    -- The payment provider's ID for the payment, like pay_29QQoUBi66xm2f.
    payment_id       text,
    updated_at       timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE orders;
