-- Extensions live in public so their types resolve for every schema on the
-- search_path. Without SCHEMA public they would land in whichever schema is
-- first on the path (a per-test schema under test isolation), and citext would
-- then be missing when another schema migrates.
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
