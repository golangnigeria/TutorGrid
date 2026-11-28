-- ============================================================================
-- Migration 005: Resource Library Table, Tenant RLS & Constrained Reconciler
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    uploaded_by_user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE RESTRICT,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 26214400),
    object_key TEXT NOT NULL UNIQUE,
    etag TEXT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'ready', 'failed', 'deleting')),
    malware_scan_status TEXT NOT NULL DEFAULT 'pending' CHECK (malware_scan_status IN ('pending', 'skipped', 'clean', 'infected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resources_tenant_status_created
    ON public.resources(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_resources_reconcile_candidates
    ON public.resources(updated_at ASC, id ASC)
    WHERE status IN ('pending', 'deleting');

ALTER TABLE public.resources ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.resources FORCE ROW LEVEL SECURITY;

-- Runtime tenant-isolated RLS policies for tutorgrid_app
DROP POLICY IF EXISTS resources_select_policy ON public.resources;
DROP POLICY IF EXISTS resources_insert_policy ON public.resources;
DROP POLICY IF EXISTS resources_update_policy ON public.resources;
DROP POLICY IF EXISTS resources_delete_policy ON public.resources;

CREATE POLICY resources_select_policy ON public.resources
    FOR SELECT
    TO tutorgrid_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY resources_insert_policy ON public.resources
    FOR INSERT
    TO tutorgrid_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY resources_update_policy ON public.resources
    FOR UPDATE
    TO tutorgrid_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY resources_delete_policy ON public.resources
    FOR DELETE
    TO tutorgrid_app
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Dedicated non-login, non-superuser, NOBYPASSRLS role owning the reconciliation pointer function
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_reconcile_definer') THEN
        CREATE ROLE tutorgrid_reconcile_definer
            NOLOGIN
            NOSUPERUSER
            NOCREATEDB
            NOCREATEROLE
            NOINHERIT
            NOREPLICATION
            NOBYPASSRLS;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO tutorgrid_reconcile_definer;
REVOKE ALL ON TABLE public.resources FROM tutorgrid_reconcile_definer;
-- Strict column-level SELECT ONLY: cannot read filename, object_key, etag, or uploaded_by_user_id
GRANT SELECT (id, tenant_id, status, created_at, updated_at)
    ON public.resources TO tutorgrid_reconcile_definer;

DROP POLICY IF EXISTS resources_reconcile_pointer_select ON public.resources;
CREATE POLICY resources_reconcile_pointer_select ON public.resources
    FOR SELECT
    TO tutorgrid_reconcile_definer
    USING (status IN ('pending', 'deleting'));

CREATE OR REPLACE FUNCTION public.resources_list_reconcile_candidates(
    p_pending_cutoff TIMESTAMPTZ,
    p_deleting_cutoff TIMESTAMPTZ,
    p_limit INT DEFAULT 50
)
RETURNS TABLE (
    tenant_id UUID,
    resource_id UUID,
    status TEXT
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
    SELECT r.tenant_id, r.id AS resource_id, r.status
    FROM public.resources r
    WHERE (r.status = 'pending' AND r.created_at <= p_pending_cutoff)
       OR (r.status = 'deleting' AND r.updated_at <= p_deleting_cutoff)
    ORDER BY r.updated_at ASC, r.id ASC
    LIMIT LEAST(GREATEST(COALESCE(p_limit, 50), 1), 100);
$$;

ALTER FUNCTION public.resources_list_reconcile_candidates(TIMESTAMPTZ, TIMESTAMPTZ, INT)
    OWNER TO tutorgrid_reconcile_definer;

REVOKE ALL ON FUNCTION public.resources_list_reconcile_candidates(TIMESTAMPTZ, TIMESTAMPTZ, INT) FROM PUBLIC;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tutorgrid_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.resources TO tutorgrid_app;
        GRANT EXECUTE ON FUNCTION public.resources_list_reconcile_candidates(TIMESTAMPTZ, TIMESTAMPTZ, INT) TO tutorgrid_app;
    END IF;
END
$$;

---- create above / drop below ----

DROP FUNCTION IF EXISTS public.resources_list_reconcile_candidates(TIMESTAMPTZ, TIMESTAMPTZ, INT);
DROP POLICY IF EXISTS resources_reconcile_pointer_select ON public.resources;
DROP POLICY IF EXISTS resources_delete_policy ON public.resources;
DROP POLICY IF EXISTS resources_update_policy ON public.resources;
DROP POLICY IF EXISTS resources_insert_policy ON public.resources;
DROP POLICY IF EXISTS resources_select_policy ON public.resources;
DROP TABLE IF EXISTS public.resources;
