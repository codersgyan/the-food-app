-- Postgres's own counters, as one line of JSON. run.sh saves them before and
-- after each step, and the difference is what the step made Postgres do:
-- how many transactions, how many pages came from memory and how many from
-- disk, and how each table was read (sequential scans read the whole table,
-- index scans jump to the rows).
SELECT json_build_object(
  'time', now(),
  'database', (
    SELECT json_build_object(
      'commits', xact_commit, 'rollbacks', xact_rollback,
      'pages_from_memory', blks_hit, 'pages_from_disk', blks_read,
      'rows_returned', tup_returned, 'rows_fetched', tup_fetched,
      'rows_inserted', tup_inserted, 'rows_updated', tup_updated,
      'deadlocks', deadlocks)
    FROM pg_stat_database WHERE datname = 'food'),
  'tables', (
    SELECT json_object_agg(relname, json_build_object(
      'seq_scans', seq_scan, 'rows_read_by_seq_scans', seq_tup_read,
      'index_scans', idx_scan, 'rows_inserted', n_tup_ins, 'rows_updated', n_tup_upd))
    FROM pg_stat_user_tables),
  'connections', (SELECT count(*) FROM pg_stat_activity WHERE datname = 'food')
);
