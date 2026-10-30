-- Each index, with and without it, on the full data (20,000 restaurants,
-- about 2 million dishes, and the 733,304 orders left by the 2-hour run).
-- Every query is the app's own query from internal/store. Each one runs once
-- to warm the cache, then once with EXPLAIN (ANALYZE, BUFFERS). The index is
-- dropped inside a transaction that is rolled back, so it is never lost.
\pset pager off
SELECT lat, lng FROM restaurants WHERE id = 777 \gset
SELECT user_id AS uid FROM addresses ORDER BY id LIMIT 1 OFFSET 5000 \gset
SELECT max(id) - 1000 AS oid FROM orders \gset

\echo '##### 1. menu: dishes(restaurant_id), WITH the index'
SELECT count(*) FROM dishes WHERE restaurant_id = 777;
EXPLAIN (ANALYZE, BUFFERS) SELECT id, restaurant_id, name, price, sold_out FROM dishes WHERE restaurant_id = 777 ORDER BY id;
\echo '##### 1. menu: dishes(restaurant_id), WITHOUT the index'
BEGIN;
DROP INDEX dishes_restaurant_id_idx;
SELECT count(*) FROM dishes WHERE restaurant_id = 777;
EXPLAIN (ANALYZE, BUFFERS) SELECT id, restaurant_id, name, price, sold_out FROM dishes WHERE restaurant_id = 777 ORDER BY id;
ROLLBACK;

\echo '##### 2. near: restaurants spatial index, WITH the index'
SELECT count(*) FROM (SELECT id FROM restaurants ORDER BY ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng, :lat)::geography LIMIT 20) x;
EXPLAIN (ANALYZE, BUFFERS) SELECT id, name, address, meters / 1000 FROM (
  SELECT id, name, address, ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng::float8, :lat::float8)::geography AS meters
  FROM restaurants
  ORDER BY ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng::float8, :lat::float8)::geography
  LIMIT 20 OFFSET 0) nearest
WHERE meters <= 7 * 1000;
\echo '##### 2. near: restaurants spatial index, WITHOUT the index'
BEGIN;
DROP INDEX restaurants_location_idx;
SELECT count(*) FROM (SELECT id FROM restaurants ORDER BY ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng, :lat)::geography LIMIT 20) x;
EXPLAIN (ANALYZE, BUFFERS) SELECT id, name, address, meters / 1000 FROM (
  SELECT id, name, address, ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng::float8, :lat::float8)::geography AS meters
  FROM restaurants
  ORDER BY ST_MakePoint(lng, lat)::geography <-> ST_MakePoint(:lng::float8, :lat::float8)::geography
  LIMIT 20 OFFSET 0) nearest
WHERE meters <= 7 * 1000;
ROLLBACK;

\echo '##### 3. tablet poll: orders(restaurant_id, status), WITH the index'
EXPLAIN (ANALYZE, BUFFERS) SELECT o.id, o.user_id, o.restaurant_id, o.placed_at, o.delivery_address, o.total, o.status, o.payment_id,
       oi.dish_id, d.name, oi.quantity, oi.price
FROM orders o JOIN order_items oi ON oi.order_id = o.id JOIN dishes d ON d.id = oi.dish_id
WHERE o.restaurant_id = 777 AND o.status = 'paid' ORDER BY o.placed_at, o.id, oi.dish_id;
\echo '##### 3. tablet poll: orders(restaurant_id, status), WITHOUT the index'
BEGIN;
DROP INDEX orders_restaurant_id_status_idx;
SELECT count(*) FROM orders WHERE restaurant_id = 777 AND status = 'paid';
EXPLAIN (ANALYZE, BUFFERS) SELECT o.id, o.user_id, o.restaurant_id, o.placed_at, o.delivery_address, o.total, o.status, o.payment_id,
       oi.dish_id, d.name, oi.quantity, oi.price
FROM orders o JOIN order_items oi ON oi.order_id = o.id JOIN dishes d ON d.id = oi.dish_id
WHERE o.restaurant_id = 777 AND o.status = 'paid' ORDER BY o.placed_at, o.id, oi.dish_id;
ROLLBACK;

\echo '##### 4. /me: addresses(user_id), WITH the index'
EXPLAIN (ANALYZE, BUFFERS) SELECT id, address FROM addresses WHERE user_id = :uid ORDER BY id;
\echo '##### 4. /me: addresses(user_id), WITHOUT the index'
BEGIN;
DROP INDEX addresses_user_id_idx;
SELECT count(*) FROM addresses WHERE user_id = :uid;
EXPLAIN (ANALYZE, BUFFERS) SELECT id, address FROM addresses WHERE user_id = :uid ORDER BY id;
ROLLBACK;

\echo '##### 5. order status: order_items(order_id), WITH the index'
EXPLAIN (ANALYZE, BUFFERS) SELECT oi.dish_id, d.name, oi.quantity, oi.price FROM order_items oi JOIN dishes d ON d.id = oi.dish_id WHERE oi.order_id = :oid ORDER BY oi.dish_id;
\echo '##### 5. order status: order_items(order_id), WITHOUT the index'
BEGIN;
DROP INDEX order_items_order_id_idx;
SELECT count(*) FROM order_items WHERE order_id = :oid;
EXPLAIN (ANALYZE, BUFFERS) SELECT oi.dish_id, d.name, oi.quantity, oi.price FROM order_items oi JOIN dishes d ON d.id = oi.dish_id WHERE oi.order_id = :oid ORDER BY oi.dish_id;
ROLLBACK;

\echo '##### table sizes'
SELECT relname, n_live_tup, pg_size_pretty(pg_relation_size(relid)) AS table_size FROM pg_stat_user_tables WHERE relname IN ('dishes','restaurants','orders','order_items','addresses') ORDER BY 1;
