import { FormEvent, useCallback, useEffect, useState } from 'react';
import {
  ApiError,
  createInvitation,
  fetchCurrentMembers,
  setActiveTenant,
  type CreatedInvitation,
  type MeResponse,
  type Role,
  type TenantMember,
} from '../../lib/api';

interface WorkspaceViewProps {
  me: MeResponse;
  onRefreshMe: () => Promise<void>;
}

export function WorkspaceView({ me, onRefreshMe }: WorkspaceViewProps) {
  const activeMembership = me.memberships.find((m) => m.tenant_id === me.active_tenant_id) ?? null;
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [membersState, setMembersState] = useState<'idle' | 'loading' | 'loaded' | 'error'>('idle');
  const [membersError, setMembersError] = useState<string | null>(null);

  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<Role>('teacher');
  const [inviteBusy, setInviteBusy] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [createdInvite, setCreatedInvite] = useState<CreatedInvitation | null>(null);
  const [switchError, setSwitchError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!activeMembership || activeMembership.role === 'learner') {
      setMembers([]);
      setMembersState('idle');
      return;
    }
    setMembersState('loading');
    setMembersError(null);
    try {
      const res = await fetchCurrentMembers();
      setMembers(res.members);
      setMembersState('loaded');
    } catch (err) {
      setMembersState('error');
      setMembersError(err instanceof ApiError ? err.message : 'Unable to load workspace members.');
    }
  }, [activeMembership]);

  useEffect(() => {
    void loadMembers();
  }, [loadMembers]);

  const handleSwitchTenant = async (tenantId: string) => {
    setSwitchError(null);
    try {
      await setActiveTenant(tenantId);
      await onRefreshMe();
    } catch (err) {
      setSwitchError(err instanceof ApiError ? err.message : 'Unable to switch teaching space.');
    }
  };

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault();
    setInviteBusy(true);
    setInviteError(null);
    setCreatedInvite(null);
    try {
      const res = await createInvitation(inviteEmail, inviteRole);
      setCreatedInvite(res.invitation);
      setInviteEmail('');
    } catch (err) {
      setInviteError(err instanceof ApiError ? err.message : 'Unable to create invitation.');
    } finally {
      setInviteBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <section className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold text-[var(--tg-ink)]">
              {activeMembership ? activeMembership.tenant_name : 'Select a Teaching Space'}
            </h1>
            <p className="mt-1 text-sm text-[var(--tg-muted)]">
              Signed in as {me.user.display_name} ({me.user.email})
              {activeMembership ? ` · Role: ${activeMembership.role}` : ''}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <label htmlFor="workspace-select" className="text-xs font-medium text-[var(--tg-muted)]">
              Teaching Space
            </label>
            <select
              id="workspace-select"
              value={me.active_tenant_id ?? ''}
              onChange={(e) => void handleSwitchTenant(e.target.value)}
              className="min-h-11 rounded-lg border border-[var(--tg-border)] bg-[var(--tg-card)] px-3 py-2 text-sm text-[var(--tg-ink)]"
            >
              {!me.active_tenant_id && <option value="">Choose space...</option>}
              {me.memberships.map((m) => (
                <option key={m.tenant_id} value={m.tenant_id}>
                  {m.tenant_name} ({m.role})
                </option>
              ))}
            </select>
          </div>
        </div>

        {switchError && (
          <p role="alert" className="mt-4 text-sm text-[var(--tg-error)]">
            {switchError}
          </p>
        )}
      </section>

      {activeMembership && activeMembership.role !== 'learner' && (
        <section className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs">
          <h2 className="text-lg font-semibold text-[var(--tg-ink)]">Workspace Members</h2>
          <p className="mt-1 text-xs text-[var(--tg-muted)]">
            Scoped server-side by verified membership and PostgreSQL Row-Level Security.
          </p>

          {membersState === 'loading' && (
            <p role="status" className="mt-4 text-sm text-[var(--tg-muted)]">
              Loading workspace members...
            </p>
          )}

          {membersState === 'error' && (
            <p role="alert" className="mt-4 text-sm text-[var(--tg-error)]">
              {membersError}
            </p>
          )}

          {membersState === 'loaded' && (
            <ul className="mt-4 divide-y divide-[var(--tg-border)]">
              {members.map((member) => (
                <li key={member.membership_id} className="flex items-center justify-between py-3 text-sm">
                  <div>
                    <p className="font-medium text-[var(--tg-ink)]">{member.display_name}</p>
                    <p className="text-xs text-[var(--tg-muted)]">{member.email}</p>
                  </div>
                  <span className="rounded-full border border-[var(--tg-border)] px-2.5 py-0.5 text-xs font-medium text-[var(--tg-muted)]">
                    {member.role}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {activeMembership && (activeMembership.role === 'org_admin' || activeMembership.role === 'platform_admin') && (
        <section className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs">
          <h2 className="text-lg font-semibold text-[var(--tg-ink)]">Invite a Member</h2>
          <form onSubmit={(e) => void handleInvite(e)} className="mt-4 grid gap-4 sm:grid-cols-3">
            <input
              type="email"
              required
              aria-label="Invitee email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder="colleague@example.com"
              className="min-h-11 rounded-lg border border-[var(--tg-border)] px-3 py-2 text-sm text-[var(--tg-ink)]"
            />
            <select
              aria-label="Invitee role"
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value as Role)}
              className="min-h-11 rounded-lg border border-[var(--tg-border)] bg-[var(--tg-card)] px-3 py-2 text-sm text-[var(--tg-ink)]"
            >
              <option value="teacher">Teacher</option>
              <option value="learner">Learner</option>
              <option value="org_admin">Organization Admin</option>
            </select>
            <button
              type="submit"
              disabled={inviteBusy}
              className="inline-flex min-h-11 items-center justify-center rounded-lg bg-[var(--tg-accent)] px-4 py-2 text-sm font-medium text-[var(--tg-accent-ink)] disabled:opacity-60"
            >
              {inviteBusy ? 'Creating Invite...' : 'Create Invitation'}
            </button>
          </form>

          {inviteError && (
            <p role="alert" className="mt-3 text-sm text-[var(--tg-error)]">
              {inviteError}
            </p>
          )}

          {createdInvite && (
            <div role="status" className="mt-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm">
              <p className="font-medium text-[var(--tg-success)]">
                Invitation created for {createdInvite.email} ({createdInvite.role})
              </p>
              <p className="mt-1 break-all font-mono text-xs text-[var(--tg-ink)]">
                Token: {createdInvite.token}
              </p>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
