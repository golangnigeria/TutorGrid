import { type FormEvent, useEffect, useState } from "react";
import { Building2, CheckCircle2, MailCheck } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import {
  AuthApiError,
  acceptMyInvitation,
  createTenant,
  fetchMyPendingInvitations,
  type AuthUser,
  type UserPendingInvitation,
} from "@/lib/auth";

interface OnboardingViewProps {
  user: AuthUser;
  onCompleted: () => Promise<void>;
}

export function OnboardingView({ user, onCompleted }: OnboardingViewProps) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const [pendingInvites, setPendingInvites] = useState<UserPendingInvitation[]>([]);
  const [loadingInvites, setLoadingInvites] = useState(true);
  const [acceptingId, setAcceptingId] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchMyPendingInvitations();
        if (!cancelled) {
          setPendingInvites(res.invitations);
        }
      } catch {
        // Non-fatal
      } finally {
        if (!cancelled) {
          setLoadingInvites(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setErrorMessage(null);
    try {
      const res = await createTenant(name, slug);
      toast.success(`Created ${res.membership.tenant_name}`);
      await onCompleted();
    } catch (err) {
      setErrorMessage(
        err instanceof AuthApiError ? err.message : "Unable to create teaching space.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleAccept = async (inv: UserPendingInvitation) => {
    setAcceptingId(inv.id);
    setErrorMessage(null);
    try {
      const res = await acceptMyInvitation(inv.id);
      toast.success(`Joined ${res.membership.tenant_name}`, {
        description: `You are now active as ${res.membership.role}.`,
      });
      await onCompleted();
    } catch (err) {
      const msg = err instanceof AuthApiError ? err.message : "Unable to accept invitation.";
      setErrorMessage(msg);
      toast.error(msg);
    } finally {
      setAcceptingId(null);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 28 }}
      className="mx-auto max-w-3xl space-y-6"
    >
      <div>
        <span className="inline-flex items-center rounded-full border border-[#D6C29F]/35 bg-[#181F1A]/78 px-2.5 py-1 text-[11px] font-medium text-[#D6C29F] backdrop-blur-md">
          Workspace Setup
        </span>
        <h1 className="mt-3 font-display text-3xl font-semibold tracking-tight text-[#F3F0E8]">
          Welcome, {user.display_name}
        </h1>
        <p className="mt-1.5 text-sm text-[#D6DDD7]">
          {pendingInvites.length > 0
            ? "Accept your pending invitation below or create your own Teaching Space."
            : "Create your own Teaching Space, or open an invitation email sent to your address to join an existing organization."}
        </p>
      </div>

      {errorMessage && (
        <p
          role="alert"
          className="rounded-xl border border-[#D97373]/40 bg-[#181F1A]/90 px-4 py-3 text-sm text-[#F4C2C2] backdrop-blur-md"
        >
          {errorMessage}
        </p>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <div className="flex flex-col justify-between rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-[#D6C29F]/35 bg-[#D6C29F]/12 text-[#D6C29F]">
                <MailCheck className="h-4 w-4" aria-hidden="true" />
              </div>
              <h2 className="font-display text-lg font-semibold text-[#F3F0E8]">
                {pendingInvites.length > 0 ? "Pending Invitations" : "Joining an Organization?"}
              </h2>
            </div>

            {loadingInvites ? (
              <p className="mt-4 text-xs text-[#A6B0A8]">Checking invitations for {user.email}...</p>
            ) : pendingInvites.length > 0 ? (
              <ul className="mt-4 space-y-3">
                {pendingInvites.map((inv) => (
                  <li
                    key={inv.id}
                    className="rounded-xl border border-[#8FAF94]/30 bg-[#101411]/80 p-4"
                  >
                    <p className="font-display text-base font-semibold text-[#F3F0E8]">
                      {inv.tenant_name}
                    </p>
                    <p className="mt-0.5 text-xs text-[#A6B0A8]">
                      Role: <span className="font-medium text-[#8FAF94]">{inv.role}</span>
                    </p>
                    <motion.button
                      whileTap={{ scale: 0.985 }}
                      type="button"
                      disabled={acceptingId !== null}
                      onClick={() => void handleAccept(inv)}
                      className="mt-3 inline-flex min-h-10 w-full cursor-pointer items-center justify-center gap-1.5 rounded-xl bg-[#8FAF94] px-4 py-2 text-xs font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60"
                    >
                      <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
                      {acceptingId === inv.id ? "Joining..." : `Accept & Join ${inv.tenant_name}`}
                    </motion.button>
                  </li>
                ))}
              </ul>
            ) : (
              <>
                <p className="mt-3 text-sm leading-relaxed text-[#A6B0A8]">
                  Ask your organization admin to send an invitation to{" "}
                  <span className="font-medium text-[#F3F0E8]">{user.email}</span>.
                </p>
                <p className="mt-3 text-xs leading-relaxed text-[#A6B0A8]">
                  Any invitation sent to this address will appear right here automatically, or you can
                  click <strong className="text-[#F3F0E8]">Accept Invitation</strong> in your email.
                </p>
              </>
            )}
          </div>

          <div className="mt-6 rounded-xl border border-[#F3F0E8]/10 bg-[#101411]/70 px-4 py-3 text-xs text-[#A6B0A8]">
            Signed in as <span className="text-[#F3F0E8]">{user.email}</span>
          </div>
        </div>

        <form
          onSubmit={(e) => void handleCreate(e)}
          className="rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 p-6 backdrop-blur-md"
        >
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-[#8FAF94]/35 bg-[#8FAF94]/12 text-[#8FAF94]">
              <Building2 className="h-4 w-4" aria-hidden="true" />
            </div>
            <h2 className="font-display text-lg font-semibold text-[#F3F0E8]">
              Create a Teaching Space
            </h2>
          </div>
          <p className="mt-2 text-xs leading-relaxed text-[#A6B0A8]">
            For independent tutors or tutoring organizations.
          </p>

          <div className="mt-5 space-y-4">
            <div>
              <label htmlFor="tenant-name" className="block text-xs font-medium text-[#E3E1D8]">
                Space Name
              </label>
              <input
                id="tenant-name"
                type="text"
                required
                minLength={2}
                maxLength={100}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Ada's Math Studio"
                className="mt-1.5 min-h-11 w-full rounded-xl border border-[#F3F0E8]/16 bg-[#101411]/85 px-3.5 py-2 text-sm text-[#F3F0E8] placeholder:text-[#7E8A80] focus-visible:border-[#8FAF94] focus-visible:outline-2 focus-visible:outline-[#8FAF94]/40"
              />
            </div>

            <div>
              <label htmlFor="tenant-slug" className="block text-xs font-medium text-[#E3E1D8]">
                Workspace Slug
              </label>
              <input
                id="tenant-slug"
                type="text"
                required
                minLength={3}
                maxLength={48}
                value={slug}
                onChange={(e) => setSlug(e.target.value.toLowerCase())}
                placeholder="e.g. adas-math-studio"
                className="mt-1.5 min-h-11 w-full rounded-xl border border-[#F3F0E8]/16 bg-[#101411]/85 px-3.5 py-2 text-sm text-[#F3F0E8] placeholder:text-[#7E8A80] focus-visible:border-[#8FAF94] focus-visible:outline-2 focus-visible:outline-[#8FAF94]/40"
              />
            </div>

            <motion.button
              whileTap={{ scale: 0.985 }}
              type="submit"
              disabled={submitting}
              className="inline-flex min-h-11 w-full cursor-pointer items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
            >
              {submitting ? "Creating..." : "Create Teaching Space"}
            </motion.button>
          </div>
        </form>
      </div>
    </motion.div>
  );
}
