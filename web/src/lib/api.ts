export type Role = 'platform_admin' | 'org_admin' | 'teacher' | 'learner';

export interface UserProfile {
  id: string;
  display_name: string;
  email: string;
}

export interface Membership {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  role: Role;
}

export interface MeResponse {
  user: UserProfile;
  memberships: Membership[];
  active_tenant_id: string | null;
}

export interface TenantMember {
  membership_id: string;
  user_id: string;
  display_name: string;
  email: string;
  role: Role;
}

export interface CreatedInvitation {
  id: string;
  email: string;
  role: Role;
  token: string;
  expires_at: string;
}

interface ErrorEnvelope {
  error?: {
    code?: string;
    message?: string;
  };
}

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
    ...init,
  });

  const payload = (await response.json().catch(() => ({}))) as T & ErrorEnvelope;
  if (!response.ok) {
    throw new ApiError(
      response.status,
      payload.error?.code ?? 'REQUEST_FAILED',
      payload.error?.message ?? `Request failed (HTTP ${response.status})`,
    );
  }
  return payload;
}

export function fetchMe(): Promise<MeResponse> {
  return request<MeResponse>('/api/v1/me');
}

export async function startOAuthSignIn(provider: 'google' | 'github'): Promise<string> {
  const origin = window.location.origin;
  const params = new URLSearchParams({
    redirect_uri: origin,
    error_redirect_uri: origin,
  });
  const payload = await request<{ url?: string; data?: { url?: string } }>(
    `/auth/oauth/${provider}/authorize?${params.toString()}`,
  );
  const url = payload.url ?? payload.data?.url;
  if (!url) {
    throw new ApiError(500, 'OAUTH_URL_MISSING', 'Auth provider did not return an authorization URL.');
  }
  return url;
}

export function signOut(): Promise<{ status: string }> {
  return request<{ status: string }>('/api/v1/auth/logout', { method: 'POST' });
}

export function createTenant(name: string, slug: string): Promise<{ membership: Membership }> {
  return request<{ membership: Membership }>('/api/v1/tenants', {
    method: 'POST',
    body: JSON.stringify({ name, slug }),
  });
}

export function setActiveTenant(tenantId: string): Promise<{ active_tenant_id: string; role: Role }> {
  return request<{ active_tenant_id: string; role: Role }>('/api/v1/session/active-tenant', {
    method: 'POST',
    body: JSON.stringify({ tenant_id: tenantId }),
  });
}

export function acceptInvitation(token: string): Promise<{ membership: Membership }> {
  return request<{ membership: Membership }>('/api/v1/invitations/accept', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
}

export function fetchCurrentMembers(): Promise<{ members: TenantMember[] }> {
  return request<{ members: TenantMember[] }>('/api/v1/tenants/current/members');
}

export function createInvitation(email: string, role: Role): Promise<{ invitation: CreatedInvitation }> {
  return request<{ invitation: CreatedInvitation }>('/api/v1/tenants/current/invitations', {
    method: 'POST',
    body: JSON.stringify({ email, role }),
  });
}
