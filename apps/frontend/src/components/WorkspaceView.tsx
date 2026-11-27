import { type FormEvent, useCallback, useEffect, useState } from "react";
import { CheckCircle2, Clock, Mail, ShieldCheck, Trash2, UserPlus, Users } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import {
  AuthApiError,
  acceptMyInvitation,
  createInvitation,
  fetchCurrentInvitations,
  fetchCurrentMembers,
  fetchMyPendingInvitations,
  revokeInvitation,
  type MeResponse,
  type PendingInvitation,
  type Role,
  type TenantMember,
  type UserPendingInvitation,
} from "@/lib/auth";

interface WorkspaceViewProps {
  me: MeResponse;
  onRefreshMe: () => Promise<void>;
}

function roleBadgeClass(role: Role): string {
  switch (role) {
    case "org_admin":
    case "platform_admin":
      return "border-[#D6C29F]/40 bg-[#D6C29F]/12 text-[#D6C29F]";
    case "teacher":
      return "border-[#8FAF94]/40 bg-[#8FAF94]/12 text-[#8FAF94]";
    default:
      return "border-[#F3F0E8]/18 bg-[#222B24] text-[#E3E1D8]";
  }
}

function emailStatusBadge(status: string): { label: string; className: string } {
  switch (status) {
    case "sent":
      return {
        label: "Email sent",
        className: "border-[#8FAF94]/35 bg-[#8FAF94]/12 text-[#8FAF94]",
      };
    case "enqueued":
      return {
        label: "Sending",
        className: "border-[#D6C29F]/35 bg-[#D6C29F]/10 text-[#D6C29F]",
      };
    case "failed":
      return {
        label: "Delivery failed",
        className: "border-[#D97373]/40 bg-[#D97373]/12 text-[#F4C2C2]",
      };
    default:
      return {
        label: "Queued",
        className: "border-[#F3F0E8]/16 bg-[#222B24] text-[#A6B0A8]",
      };
  }
}

function formatExpiry(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function WorkspaceView({ me, onRefreshMe }: WorkspaceViewProps) {
  // Always display the backend-verified active_tenant_id; switching is handled in AuthenticatedHeader.
  const activeMembership =
    me.memberships.find((m) => m.tenant_id === me.active_tenant_id) ?? null;

  const isAdmin =
    activeMembership !== null &&
    (activeMembership.role === "org_admin" ||
      activeMembership.role === "platform_admin");

  const [myPendingInvites, setMyPendingInvites] = useState<UserPendingInvitation[]>([]);
  const [acceptingMyInviteId, setAcceptingMyInviteId] = useState<string | null>(null);

  const [members, setMembers] = useState<TenantMember[]>([]);
  const [membersState, setMembersState] = useState<"idle" | "loading" | "loaded" | "error">("idle");
  const [membersError, setMembersError] = useState<string | null>(null);

  const [invitations, setInvitations] = useState<PendingInvitation[]>([]);
  const [invitationsLoading, setInvitationsLoading] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<Role>("teacher");
  const [inviteBusy, setInviteBusy] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!activeMembership || activeMembership.role === "learner") {
      setMembers([]);
      setMembersState("idle");
      return;
    }
    setMembersState("loading");
    setMembersError(null);
    try {
      const res = await fetchCurrentMembers();
      setMembers(res.members);
      setMembersState("loaded");
    } catch (err) {
      setMembersState("error");
      setMembersError(
        err instanceof AuthApiError ? err.message : "Unable to load workspace members.",
      );
    }
  }, [activeMembership]);

  const loadInvitations = useCallback(async () => {
    if (!isAdmin) {
      setInvitations([]);
      return;
    }
    setInvitationsLoading(true);
    try {
      const res = await fetchCurrentInvitations();
      setInvitations(res.invitations);
    } catch {
      // Non-fatal; admin can still create invites
    } finally {
      setInvitationsLoading(false);
    }
  }, [isAdmin]);

  useEffect(() => {
    void loadMembers();
    void loadInvitations();
    void (async () => {
      try {
        const res = await fetchMyPendingInvitations();
        setMyPendingInvites(res.invitations);
      } catch {
        // Non-fatal
      }
    })();
  }, [loadMembers, loadInvitations]);

  const handleAcceptMyInvite = async (inv: UserPendingInvitation) => {
    setAcceptingMyInviteId(inv.id);
    try {
      const res = await acceptMyInvitation(inv.id);
      toast.success(`Joined ${res.membership.tenant_name}`, {
        description: `Switched active space as ${res.membership.role}.`,
      });
      setMyPendingInvites((prev) => prev.filter((item) => item.id !== inv.id));
      await onRefreshMe();
    } catch (err) {
      toast.error(err instanceof AuthApiError ? err.message : "Unable to accept invitation.");
    } finally {
      setAcceptingMyInviteId(null);
    }
  };

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault();
    setInviteBusy(true);
    setInviteError(null);
    try {
      const res = await createInvitation(inviteEmail, inviteRole);
      setInviteEmail("");
      toast.success(`Invitation sent to ${res.invitation.email}`, {
        description: `Invited as ${res.invitation.role}. They'll receive an email link to join.`,
      });
      await loadInvitations();
    } catch (err) {
      const msg = err instanceof AuthApiError ? err.message : "Unable to send invitation.";
      setInviteError(msg);
      toast.error(msg);
    } finally {
      setInviteBusy(false);
    }
  };

  const handleRevoke = async (inv: PendingInvitation) => {
    setRevokingId(inv.id);
    try {
      await revokeInvitation(inv.id);
      setInvitations((prev) => prev.filter((item) => item.id !== inv.id));
      toast.success(`Invitation to ${inv.email} revoked`);
    } catch (err) {
      toast.error(err instanceof AuthApiError ? err.message : "Unable to revoke invitation.");
    } finally {
      setRevokingId(null);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 28 }}
      className="space-y-6"
    >
      {myPendingInvites.length > 0 && (
        <section className="rounded-2xl border border-[#D6C29F]/35 bg-[#181F1A]/86 p-5 backdrop-blur-md">
          <p className="text-xs font-semibold tracking-wider text-[#D6C29F] uppercase">
            Pending Invitations for You
          </p>
          <ul className="mt-3 space-y-2.5">
            {myPendingInvites.map((inv) => (
              <li
                key={inv.id}
                className="flex flex-col gap-3 rounded-xl border border-[#F3F0E8]/10 bg-[#101411]/75 p-3.5 sm:flex-row sm:items-center sm:justify-between"
              >
                <div>
                  <p className="font-display text-sm font-semibold text-[#F3F0E8]">
                    {inv.tenant_name}
                  </p>
                  <p className="text-xs text-[#A6B0A8]">
                    Invited as <span className="text-[#8FAF94]">{inv.role}</span>
                  </p>
                </div>
                <button
                  type="button"
                  disabled={acceptingMyInviteId !== null}
                  onClick={() => void handleAcceptMyInvite(inv)}
                  className="inline-flex min-h-9 cursor-pointer items-center justify-center gap-1.5 rounded-xl bg-[#8FAF94] px-3.5 py-1.5 text-xs font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60"
                >
                  <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
                  {acceptingMyInviteId === inv.id ? "Joining..." : "Accept & Join"}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
        <div className="flex flex-wrap items-center gap-2.5">
          <h1 className="font-display text-2xl font-semibold tracking-tight text-[#F3F0E8]">
            {activeMembership ? activeMembership.tenant_name : "Select a Teaching Space"}
          </h1>
          {activeMembership && (
            <span
              className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-[11px] font-medium ${roleBadgeClass(activeMembership.role)}`}
            >
              <ShieldCheck className="h-3 w-3" aria-hidden="true" />
              {activeMembership.role}
            </span>
          )}
        </div>
        <p className="mt-1.5 text-sm text-[#A6B0A8]">
          {activeMembership
            ? `Active teaching space (${activeMembership.tenant_slug}). Switch spaces anytime from the top workspace pill.`
            : "Use the Teaching Spaces switcher in the top left pill to select an active workspace."}
        </p>
      </section>

      {activeMembership && activeMembership.role !== "learner" && (
        <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
          <div className="flex items-center gap-2.5">
            <Users className="h-4 w-4 text-[#8FAF94]" aria-hidden="true" />
            <h2 className="text-base font-semibold text-[#F3F0E8]">Workspace Members</h2>
          </div>
          <p className="mt-1 text-xs text-[#A6B0A8]">
            Enforced server-side via verified membership and PostgreSQL Dual RLS.
          </p>

          {membersState === "loading" && (
            <p role="status" className="mt-4 text-sm text-[#A6B0A8]">
              Loading workspace members...
            </p>
          )}

          {membersState === "error" && (
            <p role="alert" className="mt-4 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-3.5 py-2 text-sm text-[#F4C2C2]">
              {membersError}
            </p>
          )}

          {membersState === "loaded" && (
            <ul className="mt-4 divide-y divide-[#F3F0E8]/10">
              {members.map((member) => (
                <li
                  key={member.membership_id}
                  className="flex items-center justify-between py-3.5 text-sm"
                >
                  <div>
                    <p className="font-medium text-[#F3F0E8]">{member.display_name}</p>
                    <p className="text-xs text-[#A6B0A8]">{member.email}</p>
                  </div>
                  <span
                    className={`rounded-full border px-2.5 py-0.5 text-xs font-medium ${roleBadgeClass(member.role)}`}
                  >
                    {member.role}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

      {isAdmin && (
        <section className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
          <div className="flex items-center gap-2.5">
            <UserPlus className="h-4 w-4 text-[#8FAF94]" aria-hidden="true" />
            <h2 className="text-base font-semibold text-[#F3F0E8]">Invite a Member</h2>
          </div>
          <p className="mt-1 text-xs text-[#A6B0A8]">
            Send a one-time email invitation restricted to the recipient&apos;s address.
          </p>

          <form
            onSubmit={(e) => void handleInvite(e)}
            className="mt-5 grid gap-3.5 sm:grid-cols-3"
          >
            <input
              type="email"
              required
              aria-label="Invitee email"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              placeholder="colleague@example.com"
              className="min-h-11 rounded-xl border border-[#F3F0E8]/16 bg-[#101411]/85 px-3.5 py-2 text-sm text-[#F3F0E8] placeholder:text-[#7E8A80] focus-visible:border-[#8FAF94] focus-visible:outline-2 focus-visible:outline-[#8FAF94]/40"
            />
            <select
              aria-label="Invitee role"
              value={inviteRole}
              onChange={(e) => setInviteRole(e.target.value as Role)}
              className="min-h-11 rounded-xl border border-[#F3F0E8]/16 bg-[#101411] px-3.5 py-2 text-sm text-[#F3F0E8] focus-visible:border-[#8FAF94] focus-visible:outline-2 focus-visible:outline-[#8FAF94]/40"
            >
              <option value="teacher">Teacher</option>
              <option value="learner">Learner</option>
              <option value="org_admin">Organization Admin</option>
            </select>
            <motion.button
              whileTap={{ scale: 0.985 }}
              type="submit"
              disabled={inviteBusy}
              className="inline-flex min-h-11 cursor-pointer items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
            >
              {inviteBusy ? "Sending Invite..." : "Send Invitation"}
            </motion.button>
          </form>

          {inviteError && (
            <p role="alert" className="mt-3 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-3.5 py-2 text-sm text-[#F4C2C2]">
              {inviteError}
            </p>
          )}

          {/* Pending Invitations Management List */}
          <div className="mt-6 border-t border-[#F3F0E8]/10 pt-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Mail className="h-3.5 w-3.5 text-[#D6C29F]" aria-hidden="true" />
                <h3 className="text-xs font-semibold tracking-wider text-[#D6C29F] uppercase">
                  Pending Invitations
                </h3>
              </div>
              {invitationsLoading && (
                <span className="text-xs text-[#A6B0A8]">Refreshing...</span>
              )}
            </div>

            {invitations.length === 0 ? (
              <p className="mt-3 text-xs text-[#A6B0A8]">
                No pending invitations for this teaching space.
              </p>
            ) : (
              <ul className="mt-3 divide-y divide-[#F3F0E8]/10">
                {invitations.map((inv) => {
                  const statusInfo = emailStatusBadge(inv.email_status);
                  return (
                    <li
                      key={inv.id}
                      className="flex flex-col gap-2 py-3 text-sm sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div className="min-w-0">
                        <p className="truncate font-medium text-[#F3F0E8]">{inv.email}</p>
                        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[#A6B0A8]">
                          <span
                            className={`rounded-full border px-2 py-0.5 text-[11px] font-medium ${roleBadgeClass(inv.role)}`}
                          >
                            {inv.role}
                          </span>
                          <span
                            className={`rounded-full border px-2 py-0.5 text-[11px] font-medium ${statusInfo.className}`}
                          >
                            {statusInfo.label}
                          </span>
                          <span className="inline-flex items-center gap-1 text-[11px] text-[#A6B0A8]">
                            <Clock className="h-3 w-3" aria-hidden="true" />
                            Expires {formatExpiry(inv.expires_at)}
                          </span>
                        </div>
                      </div>

                      <button
                        type="button"
                        disabled={revokingId === inv.id}
                        onClick={() => void handleRevoke(inv)}
                        className="inline-flex min-h-9 shrink-0 cursor-pointer items-center gap-1.5 self-start rounded-lg border border-[#D97373]/30 bg-[#D97373]/10 px-3 py-1.5 text-xs font-medium text-[#F4C2C2] transition-colors hover:bg-[#D97373]/20 disabled:opacity-50 sm:self-center"
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                        {revokingId === inv.id ? "Revoking..." : "Revoke"}
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </section>
      )}
    </motion.div>
  );
}
