-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- 1. Limen Authentication Tables (prefixed limen_* to avoid domain collisions)
-- ============================================================================
CREATE TABLE limen_users (
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

CREATE TABLE limen_sessions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    token TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL REFERENCES limen_users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_access TIMESTAMPTZ NOT NULL,
    metadata JSONB NULL,
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_limen_sessions_user_id ON limen_sessions(user_id);

CREATE TABLE limen_accounts (
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
CREATE INDEX idx_limen_accounts_user_id ON limen_accounts(user_id);

CREATE TABLE limen_verifications (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    subject TEXT NOT NULL,
    value TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_limen_verifications_subject ON limen_verifications(subject);

CREATE TABLE limen_rate_limits (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    key TEXT NOT NULL UNIQUE,
    count INTEGER NOT NULL,
    last_request_at BIGINT NOT NULL
);

-- ============================================================================
-- 2. Server-Bound Active Tenant Selection & RLS-Protected Invitations
-- ============================================================================
CREATE TABLE user_active_tenants (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    active_tenant_id UUID NULL REFERENCES tenants(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('org_admin', 'teacher', 'learner')),
    token_hash BYTEA NOT NULL UNIQUE,
    invited_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tenant_invitations_tenant_id ON tenant_invitations(tenant_id);

ALTER TABLE tenant_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_invitations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_invitations_tenant_isolation ON tenant_invitations
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- ============================================================================
-- 3. Narrow SECURITY DEFINER Pre-Tenant Identity & Membership Lookup Functions
--    Allows tutorgrid_app (NOBYPASSRLS) to resolve verified user memberships
--    without relaxing FORCE ROW LEVEL SECURITY on tenant-scoped tables.
-- ============================================================================
CREATE OR REPLACE FUNCTION auth_upsert_user_by_subject(
    p_external_subject TEXT,
    p_display_name TEXT,
    p_email TEXT
)
RETURNS TABLE (
    id UUID,
    external_subject TEXT,
    display_name TEXT,
    email TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
#variable_conflict use_column
BEGIN
    INSERT INTO users (external_subject, display_name, email)
    VALUES (p_external_subject, p_display_name, LOWER(TRIM(p_email)))
    ON CONFLICT (external_subject) DO UPDATE
        SET display_name = EXCLUDED.display_name,
            email = EXCLUDED.email,
            updated_at = NOW();

    RETURN QUERY
    SELECT u.id, u.external_subject, u.display_name, u.email
    FROM users u
    WHERE u.external_subject = p_external_subject;
END;
$$;

CREATE OR REPLACE FUNCTION auth_list_user_memberships(p_user_id UUID)
RETURNS TABLE (
    tenant_id UUID,
    tenant_name TEXT,
    tenant_slug TEXT,
    role TEXT
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT t.id, t.name, t.slug, tm.role
    FROM tenant_memberships tm
    INNER JOIN tenants t ON t.id = tm.tenant_id
    WHERE tm.user_id = p_user_id
    ORDER BY t.created_at ASC;
$$;

CREATE OR REPLACE FUNCTION auth_verify_user_membership(p_user_id UUID, p_tenant_id UUID)
RETURNS TABLE (role TEXT)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT tm.role
    FROM tenant_memberships tm
    WHERE tm.user_id = p_user_id
      AND tm.tenant_id = p_tenant_id
    LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION auth_get_active_tenant(p_user_id UUID)
RETURNS UUID
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT uat.active_tenant_id
    FROM user_active_tenants uat
    INNER JOIN tenant_memberships tm
        ON tm.user_id = uat.user_id
       AND tm.tenant_id = uat.active_tenant_id
    WHERE uat.user_id = p_user_id
    LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION auth_set_active_tenant(p_user_id UUID, p_tenant_id UUID)
RETURNS TABLE (tenant_id UUID, role TEXT)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    v_role TEXT;
BEGIN
    SELECT tm.role INTO v_role
    FROM tenant_memberships tm
    WHERE tm.user_id = p_user_id
      AND tm.tenant_id = p_tenant_id;

    IF v_role IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
    VALUES (p_user_id, p_tenant_id, NOW())
    ON CONFLICT (user_id) DO UPDATE
        SET active_tenant_id = EXCLUDED.active_tenant_id,
            updated_at = NOW();

    RETURN QUERY SELECT p_tenant_id, v_role;
END;
$$;

CREATE OR REPLACE FUNCTION auth_create_tenant_with_admin(
    p_user_id UUID,
    p_name TEXT,
    p_slug TEXT
)
RETURNS TABLE (
    tenant_id UUID,
    tenant_name TEXT,
    tenant_slug TEXT,
    role TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    v_tenant_id UUID;
BEGIN
    INSERT INTO tenants (name, slug)
    VALUES (TRIM(p_name), LOWER(TRIM(p_slug)))
    RETURNING id INTO v_tenant_id;

    INSERT INTO tenant_memberships (tenant_id, user_id, role)
    VALUES (v_tenant_id, p_user_id, 'org_admin');

    INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
    VALUES (p_user_id, v_tenant_id, NOW())
    ON CONFLICT (user_id) DO UPDATE
        SET active_tenant_id = EXCLUDED.active_tenant_id,
            updated_at = NOW();

    RETURN QUERY
    SELECT v_tenant_id, TRIM(p_name), LOWER(TRIM(p_slug)), 'org_admin'::TEXT;
END;
$$;

CREATE OR REPLACE FUNCTION auth_accept_tenant_invitation(
    p_user_id UUID,
    p_token_hash BYTEA
)
RETURNS TABLE (
    tenant_id UUID,
    tenant_name TEXT,
    tenant_slug TEXT,
    role TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
#variable_conflict use_column
DECLARE
    v_inv RECORD;
    v_tenant RECORD;
BEGIN
    SELECT ti.id, ti.tenant_id, ti.role
    INTO v_inv
    FROM tenant_invitations ti
    WHERE ti.token_hash = p_token_hash
      AND ti.accepted_at IS NULL
      AND ti.expires_at > NOW()
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    INSERT INTO tenant_memberships (tenant_id, user_id, role)
    VALUES (v_inv.tenant_id, p_user_id, v_inv.role)
    ON CONFLICT (tenant_id, user_id) DO NOTHING;

    UPDATE tenant_invitations
    SET accepted_at = NOW()
    WHERE id = v_inv.id;

    INSERT INTO user_active_tenants (user_id, active_tenant_id, updated_at)
    VALUES (p_user_id, v_inv.tenant_id, NOW())
    ON CONFLICT (user_id) DO UPDATE
        SET active_tenant_id = EXCLUDED.active_tenant_id,
            updated_at = NOW();

    SELECT t.id, t.name, t.slug INTO v_tenant
    FROM tenants t
    WHERE t.id = v_inv.tenant_id;

    RETURN QUERY
    SELECT v_tenant.id, v_tenant.name, v_tenant.slug, v_inv.role;
END;
$$;

-- Restrict function execution strictly to tutorgrid_app (revoke PUBLIC).
REVOKE ALL ON FUNCTION auth_upsert_user_by_subject(TEXT, TEXT, TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_list_user_memberships(UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_verify_user_membership(UUID, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_get_active_tenant(UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_set_active_tenant(UUID, UUID) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_create_tenant_with_admin(UUID, TEXT, TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION auth_accept_tenant_invitation(UUID, BYTEA) FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            limen_users,
            limen_sessions,
            limen_accounts,
            limen_verifications,
            limen_rate_limits,
            tenant_invitations
        TO tutorgrid_app;

        GRANT EXECUTE ON FUNCTION auth_upsert_user_by_subject(TEXT, TEXT, TEXT) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_list_user_memberships(UUID) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_verify_user_membership(UUID, UUID) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_get_active_tenant(UUID) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_set_active_tenant(UUID, UUID) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_create_tenant_with_admin(UUID, TEXT, TEXT) TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION auth_accept_tenant_invitation(UUID, BYTEA) TO tutorgrid_app;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS auth_accept_tenant_invitation(UUID, BYTEA);
DROP FUNCTION IF EXISTS auth_create_tenant_with_admin(UUID, TEXT, TEXT);
DROP FUNCTION IF EXISTS auth_set_active_tenant(UUID, UUID);
DROP FUNCTION IF EXISTS auth_get_active_tenant(UUID);
DROP FUNCTION IF EXISTS auth_verify_user_membership(UUID, UUID);
DROP FUNCTION IF EXISTS auth_list_user_memberships(UUID);
DROP FUNCTION IF EXISTS auth_upsert_user_by_subject(TEXT, TEXT, TEXT);

DROP POLICY IF EXISTS tenant_invitations_tenant_isolation ON tenant_invitations;
DROP TABLE IF EXISTS tenant_invitations;
DROP TABLE IF EXISTS user_active_tenants;

DROP TABLE IF EXISTS limen_rate_limits;
DROP TABLE IF EXISTS limen_verifications;
DROP TABLE IF EXISTS limen_accounts;
DROP TABLE IF EXISTS limen_sessions;
DROP TABLE IF EXISTS limen_users;
-- +goose StatementEnd
