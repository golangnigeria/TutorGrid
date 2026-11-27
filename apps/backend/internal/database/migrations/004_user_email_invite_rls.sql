-- ============================================================================
-- Migration 004: Verified-Email Pending Invitation Discovery & Acceptance RLS
-- ============================================================================
-- Allows an authenticated user (app.current_user_id + app.current_user_email
-- set server-side from verified OAuth identity) to view and accept pending
-- invitations addressed to their exact email without SECURITY DEFINER.

DROP POLICY IF EXISTS tenant_invitations_select_policy ON tenant_invitations;
CREATE POLICY tenant_invitations_select_policy ON tenant_invitations
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND NULLIF(current_setting('app.current_user_email', true), '') IS NOT NULL
            AND LOWER(email) = LOWER(current_setting('app.current_user_email', true))
        )
    );

DROP POLICY IF EXISTS tenant_invitations_update_policy ON tenant_invitations;
CREATE POLICY tenant_invitations_update_policy ON tenant_invitations
    FOR UPDATE
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND NULLIF(current_setting('app.current_user_email', true), '') IS NOT NULL
            AND LOWER(email) = LOWER(current_setting('app.current_user_email', true))
        )
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND NULLIF(current_setting('app.current_user_email', true), '') IS NOT NULL
            AND LOWER(email) = LOWER(current_setting('app.current_user_email', true))
        )
    );

DROP POLICY IF EXISTS tenants_select_policy ON tenants;
CREATE POLICY tenants_select_policy ON tenants
    FOR SELECT
    USING (
        id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR EXISTS (
            SELECT 1 FROM tenant_memberships tm
            WHERE tm.tenant_id = tenants.id
              AND tm.user_id = NULLIF(current_setting('app.current_user_id', true), '')::uuid
        )
        OR EXISTS (
            SELECT 1 FROM tenant_invitations ti
            WHERE ti.tenant_id = tenants.id
              AND ti.accepted_at IS NULL
              AND ti.revoked_at IS NULL
              AND ti.expires_at > NOW()
              AND NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
              AND NULLIF(current_setting('app.current_user_email', true), '') IS NOT NULL
              AND LOWER(ti.email) = LOWER(current_setting('app.current_user_email', true))
        )
    );

---- create above / drop below ----

DROP POLICY IF EXISTS tenants_select_policy ON tenants;
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

DROP POLICY IF EXISTS tenant_invitations_update_policy ON tenant_invitations;
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

DROP POLICY IF EXISTS tenant_invitations_select_policy ON tenant_invitations;
CREATE POLICY tenant_invitations_select_policy ON tenant_invitations
    FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid
        OR (
            NULLIF(current_setting('app.current_user_id', true), '')::uuid IS NOT NULL
            AND token_hash = decode(NULLIF(current_setting('app.current_invite_token_hash', true), ''), 'hex')
        )
    );
