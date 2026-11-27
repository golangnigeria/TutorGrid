-- ============================================================================
-- Migration 003: Dual RLS Policies, Active Workspace Selection,
--                Tenant Invitations & Transactional Email Outbox
-- ============================================================================

-- 1. Server-bound active workspace per user (strictly user-scoped RLS)
CREATE TABLE IF NOT EXISTS user_active_tenants (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    active_tenant_id UUID NULL REFERENCES tenants(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE user_active_tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_active_tenants FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_active_tenants_user_isolation ON user_active_tenants;
CREATE POLICY user_active_tenants_user_isolation ON user_active_tenants
    FOR ALL
    USING (user_id = NULLIF(current_setting('app.current_user_id', true), '')::uuid)
    WITH CHECK (user_id = NULLIF(current_setting('app.current_user_id', true), '')::uuid);

-- 2. Refine tenants RLS policies (per-operation, never blanket)
DROP POLICY IF EXISTS tenants_self_isolation ON tenants;
DROP POLICY IF EXISTS tenants_select_policy ON tenants;
DROP POLICY IF EXISTS tenants_insert_policy ON tenants;
DROP POLICY IF EXISTS tenants_update_policy ON tenants;
DROP POLICY IF EXISTS tenants_delete_policy ON tenants;

CREATE POLICY tenants_select_policy ON tenants
    FOR SELECT
    USING (
        id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR EXISTS (
            SELECT 1 FROM tenant_memberships tm
            WHERE tm.tenant_id = tenants.id
              AND tm.user_id = NULLIF(current_setting('app.current_user_id', true), '')::uuid
        )
    );

CREATE POLICY tenants_insert_policy ON tenants
    FOR INSERT
    WITH CHECK (NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL);

CREATE POLICY tenants_update_policy ON tenants
    FOR UPDATE
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY tenants_delete_policy ON tenants
    FOR DELETE
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 3. Refine tenant_memberships Dual RLS policies (SELECT dual, mutations strictly tenant-scoped)
DROP POLICY IF EXISTS tenant_memberships_tenant_isolation ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_select_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_insert_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_update_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_delete_policy ON tenant_memberships;

CREATE POLICY tenant_memberships_select_policy ON tenant_memberships
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR user_id = NULLIF(current_setting('app.current_user_id', true), '')::uuid
    );

CREATE POLICY tenant_memberships_insert_policy ON tenant_memberships
    FOR INSERT
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY tenant_memberships_update_policy ON tenant_memberships
    FOR UPDATE
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY tenant_memberships_delete_policy ON tenant_memberships
    FOR DELETE
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 4. Tenant invitations (tenant-scoped + narrow token-hash acceptance scope)
CREATE TABLE IF NOT EXISTS tenant_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('org_admin', 'teacher', 'learner')),
    token_hash BYTEA NOT NULL UNIQUE,
    invited_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE tenant_invitations ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ NULL;
ALTER TABLE tenant_invitations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
CREATE INDEX IF NOT EXISTS idx_tenant_invitations_tenant_id ON tenant_invitations(tenant_id);

ALTER TABLE tenant_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_invitations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_invitations_select_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_insert_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_update_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_delete_policy ON tenant_invitations;

CREATE POLICY tenant_invitations_select_policy ON tenant_invitations
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
    );

CREATE POLICY tenant_invitations_insert_policy ON tenant_invitations
    FOR INSERT
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY tenant_invitations_update_policy ON tenant_invitations
    FOR UPDATE
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
    );

CREATE POLICY tenant_invitations_delete_policy ON tenant_invitations
    FOR DELETE
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5. Transactional Outbox for durable invitation email delivery
CREATE TABLE IF NOT EXISTS invitation_email_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id UUID NOT NULL UNIQUE REFERENCES tenant_invitations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL,
    tenant_name TEXT NOT NULL,
    role TEXT NOT NULL,
    invite_url TEXT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'enqueued', 'sent', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    sent_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invitation_email_outbox_pending
    ON invitation_email_outbox(tenant_id, status, created_at)
    WHERE status IN ('pending', 'failed');

ALTER TABLE invitation_email_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE invitation_email_outbox FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS invitation_email_outbox_tenant_isolation ON invitation_email_outbox;
CREATE POLICY invitation_email_outbox_tenant_isolation ON invitation_email_outbox
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Minimal runtime grants to non-superuser tutorgrid_app (no ownership, no TRUNCATE)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON
            user_active_tenants,
            tenant_invitations,
            invitation_email_outbox
        TO tutorgrid_app;
    END IF;
END
$$;

---- create above / drop below ----

DROP POLICY IF EXISTS invitation_email_outbox_tenant_isolation ON invitation_email_outbox;
DROP TABLE IF EXISTS invitation_email_outbox;

DROP POLICY IF EXISTS tenant_invitations_delete_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_update_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_insert_policy ON tenant_invitations;
DROP POLICY IF EXISTS tenant_invitations_select_policy ON tenant_invitations;
DROP TABLE IF EXISTS tenant_invitations;

DROP POLICY IF EXISTS user_active_tenants_user_isolation ON user_active_tenants;
DROP TABLE IF EXISTS user_active_tenants;

-- Restore strict tenant-only policies from migration 002
DROP POLICY IF EXISTS tenant_memberships_delete_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_update_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_insert_policy ON tenant_memberships;
DROP POLICY IF EXISTS tenant_memberships_select_policy ON tenant_memberships;
CREATE POLICY tenant_memberships_tenant_isolation ON tenant_memberships
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS tenants_delete_policy ON tenants;
DROP POLICY IF EXISTS tenants_update_policy ON tenants;
DROP POLICY IF EXISTS tenants_insert_policy ON tenants;
DROP POLICY IF EXISTS tenants_select_policy ON tenants;
CREATE POLICY tenants_self_isolation ON tenants
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
