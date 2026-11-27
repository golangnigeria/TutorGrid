-- Create non-superuser application role so PostgreSQL RLS is strictly enforced.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_app') THEN
        CREATE ROLE tutorgrid_app
            LOGIN
            NOSUPERUSER
            NOCREATEDB
            NOCREATEROLE
            NOINHERIT
            NOREPLICATION
            NOBYPASSRLS
            PASSWORD 'tutorgrid_app_dev';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE tutorgrid TO tutorgrid_app;
GRANT USAGE ON SCHEMA public TO tutorgrid_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tutorgrid_migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tutorgrid_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tutorgrid_migrator IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO tutorgrid_app;

CREATE DATABASE tutorgrid_test OWNER tutorgrid_migrator;
GRANT CONNECT ON DATABASE tutorgrid_test TO tutorgrid_app;

\connect tutorgrid_test

GRANT USAGE ON SCHEMA public TO tutorgrid_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tutorgrid_migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO tutorgrid_app;
ALTER DEFAULT PRIVILEGES FOR ROLE tutorgrid_migrator IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO tutorgrid_app;
