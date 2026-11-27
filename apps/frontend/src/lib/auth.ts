import { API_URL } from "@/config/env";

export type Role = "platform_admin" | "org_admin" | "teacher" | "learner";

export interface AuthUser {
  id: string;
  email: string;
  display_name: string;
}

export interface Membership {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  role: Role;
}

export interface MeResponse {
  user: AuthUser;
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
  email_status: string;
  expires_at: string;
}

export interface PendingInvitation {
  id: string;
  email: string;
  role: Role;
  email_status: string;
  expires_at: string;
  created_at: string;
}

export interface UserPendingInvitation {
  id: string;
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  role: Role;
  expires_at: string;
  created_at: string;
}

export class AuthApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
    ...init,
  });

  const payload = (await response.json().catch(() => ({}))) as T & {
    code?: string;
    message?: string;
    error?: { code?: string; message?: string };
  };

  if (!response.ok) {
    throw new AuthApiError(
      response.status,
      payload.error?.code ?? payload.code ?? "REQUEST_FAILED",
      payload.error?.message ?? payload.message ?? `Request failed (HTTP ${response.status})`,
    );
  }

  return payload;
}

export function fetchMe(): Promise<MeResponse> {
  return requestJSON<MeResponse>("/api/v1/me");
}

export async function startOAuthSignIn(
  provider: "google" | "github",
  redirectPath = "/dashboard",
): Promise<string> {
  const origin = window.location.origin;
  const safePath = redirectPath.startsWith("/") ? redirectPath : "/dashboard";
  const params = new URLSearchParams({
    redirect_uri: `${origin}${safePath}`,
    error_redirect_uri: `${origin}/sign-in`,
  });

  const payload = await requestJSON<{ url?: string; data?: { url?: string } }>(
    `/auth/oauth/${provider}/authorize?${params.toString()}`,
  );

  const url = payload.url ?? payload.data?.url;
  if (!url) {
    throw new AuthApiError(500, "OAUTH_URL_MISSING", "Limen did not return an OAuth authorization URL.");
  }
  return url;
}

export function signOut(): Promise<{ status: string }> {
  return requestJSON<{ status: string }>("/api/v1/auth/logout", {
    method: "POST",
  });
}

export function createTenant(name: string, slug: string): Promise<{ membership: Membership }> {
  return requestJSON<{ membership: Membership }>("/api/v1/tenants", {
    method: "POST",
    body: JSON.stringify({ name, slug }),
  });
}

export function setActiveTenant(tenantId: string): Promise<{ active_tenant_id: string; role: Role }> {
  return requestJSON<{ active_tenant_id: string; role: Role }>("/api/v1/session/active-tenant", {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId }),
  });
}

export function acceptInvitation(token: string): Promise<{ membership: Membership }> {
  return requestJSON<{ membership: Membership }>("/api/v1/invitations/accept", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export function fetchMyPendingInvitations(): Promise<{ invitations: UserPendingInvitation[] }> {
  return requestJSON<{ invitations: UserPendingInvitation[] }>("/api/v1/me/invitations");
}

export function acceptMyInvitation(invitationId: string): Promise<{ membership: Membership }> {
  return requestJSON<{ membership: Membership }>(
    `/api/v1/me/invitations/${encodeURIComponent(invitationId)}/accept`,
    { method: "POST" },
  );
}

export function fetchCurrentMembers(): Promise<{ members: TenantMember[] }> {
  return requestJSON<{ members: TenantMember[] }>("/api/v1/tenants/current/members");
}

export function fetchCurrentInvitations(): Promise<{ invitations: PendingInvitation[] }> {
  return requestJSON<{ invitations: PendingInvitation[] }>("/api/v1/tenants/current/invitations");
}

export function createInvitation(
  email: string,
  role: Role,
): Promise<{ invitation: CreatedInvitation }> {
  return requestJSON<{ invitation: CreatedInvitation }>("/api/v1/tenants/current/invitations", {
    method: "POST",
    body: JSON.stringify({ email, role }),
  });
}

export function revokeInvitation(invitationId: string): Promise<{ status: string }> {
  return requestJSON<{ status: string }>(
    `/api/v1/tenants/current/invitations/${encodeURIComponent(invitationId)}`,
    { method: "DELETE" },
  );
}
