-- +goose Up
-- GET /v1/restaurants?near=… finds the restaurants within a few km of a
-- customer. A sorted (B-tree) index can't answer "what's near this point?",
-- so without this spatial index, Postgres works out the distance to every
-- restaurant on every call. The index is on the point made from lat and lng,
-- so the table needs no new column, and store.RestaurantsNear must use
-- exactly this expression for Postgres to use it.
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE INDEX restaurants_location_idx ON restaurants USING gist ((ST_MakePoint(lng, lat)::geography));

-- +goose Down
DROP INDEX restaurants_location_idx;
DROP EXTENSION postgis;
