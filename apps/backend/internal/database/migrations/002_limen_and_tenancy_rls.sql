-- Limen authentication tables (prefixed limen_* to avoid domain collisions)
CREATE TABLE IF NOT EXISTS limen_users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email TEXT NOT NULL UNIQUE,
    password TEXT NULL,
    email_verified_at TIMESTAMPTZ NULL,
    first_name TEXT NULL,
    last_name TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS limen_sessions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    token TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL REFERENCES limen_users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_access TIMESTAMPTZ NOT NULL,
    metadata JSONB NULL,
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_limen_sessions_user_id ON limen_sessions(user_id);

CREATE TABLE IF NOT EXISTS limen_accounts (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES limen_users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    provider TEXT NOT NULL,
    provider_account_id TEXT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NULL,
    access_token_expires_at TIMESTAMPTZ NULL,
    scope TEXT NOT NULL,
    id_token TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (provider, provider_account_id)
);
CREATE INDEX IF NOT EXISTS idx_limen_accounts_user_id ON limen_accounts(user_id);

CREATE TABLE IF NOT EXISTS limen_verifications (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    subject TEXT NOT NULL,
    value TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_limen_verifications_subject ON limen_verifications(subject);

CREATE TABLE IF NOT EXISTS limen_rate_limits (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    key TEXT NOT NULL UNIQUE,
    count INTEGER NOT NULL,
    last_request_at BIGINT NOT NULL
);

-- TutorGrid tenancy and RLS foundation
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_subject TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('platform_admin', 'org_admin', 'teacher', 'learner')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant_id ON tenant_memberships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_memberships_user_id ON tenant_memberships(user_id);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenants_self_isolation ON tenants;
CREATE POLICY tenants_self_isolation ON tenants
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE tenant_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_memberships FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_memberships_tenant_isolation ON tenant_memberships;
CREATE POLICY tenant_memberships_tenant_isolation ON tenant_memberships
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            limen_users, limen_sessions, limen_accounts, limen_verifications, limen_rate_limits,
            tenants, users, tenant_memberships
        TO tutorgrid_app;
    END IF;
END
$$;

---- create above / drop below ----

DROP POLICY IF EXISTS tenant_memberships_tenant_isolation ON tenant_memberships;
DROP POLICY IF EXISTS tenants_self_isolation ON tenants;
DROP TABLE IF EXISTS tenant_memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
DROP TABLE IF EXISTS limen_rate_limits;
DROP TABLE IF EXISTS limen_verifications;
DROP TABLE IF EXISTS limen_accounts;
DROP TABLE IF EXISTS limen_sessions;
DROP TABLE IF EXISTS limen_users;
