import { FormEvent, useState } from 'react';
import { ApiError, acceptInvitation, createTenant, type UserProfile } from '../../lib/api';

interface OnboardingViewProps {
  user: UserProfile;
  onCompleted: () => Promise<void>;
}

export function OnboardingView({ user, onCompleted }: OnboardingViewProps) {
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [inviteToken, setInviteToken] = useState('');
  const [submitting, setSubmitting] = useState<'create' | 'invite' | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting('create');
    setErrorMessage(null);
    try {
      await createTenant(name, slug);
      await onCompleted();
    } catch (err) {
      setErrorMessage(err instanceof ApiError ? err.message : 'Unable to create teaching space.');
    } finally {
      setSubmitting(null);
    }
  };

  const handleAcceptInvite = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting('invite');
    setErrorMessage(null);
    try {
      await acceptInvitation(inviteToken);
      await onCompleted();
    } catch (err) {
      setErrorMessage(err instanceof ApiError ? err.message : 'Unable to accept invitation.');
    } finally {
      setSubmitting(null);
    }
  };

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-[var(--tg-ink)]">
          Welcome, {user.display_name}
        </h1>
        <p className="mt-1 text-sm text-[var(--tg-muted)]">
          Create your own Teaching Space or accept an invitation from an organization to get started.
        </p>
      </div>

      {errorMessage && (
        <p
          role="alert"
          className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-[var(--tg-error)]"
        >
          {errorMessage}
        </p>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <form
          onSubmit={(e) => void handleCreate(e)}
          className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs"
        >
          <h2 className="text-lg font-semibold text-[var(--tg-ink)]">Create a Teaching Space</h2>
          <p className="mt-1 text-xs text-[var(--tg-muted)]">
            For independent tutors or tutoring organizations.
          </p>

          <div className="mt-4 space-y-4">
            <div>
              <label htmlFor="tenant-name" className="block text-sm font-medium text-[var(--tg-ink)]">
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
                className="mt-1 min-h-11 w-full rounded-lg border border-[var(--tg-border)] px-3 py-2 text-sm text-[var(--tg-ink)] focus-visible:outline-2 focus-visible:outline-[var(--tg-accent)]"
              />
            </div>

            <div>
              <label htmlFor="tenant-slug" className="block text-sm font-medium text-[var(--tg-ink)]">
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
                className="mt-1 min-h-11 w-full rounded-lg border border-[var(--tg-border)] px-3 py-2 text-sm text-[var(--tg-ink)] focus-visible:outline-2 focus-visible:outline-[var(--tg-accent)]"
              />
            </div>

            <button
              type="submit"
              disabled={submitting !== null}
              className="inline-flex min-h-11 w-full items-center justify-center rounded-lg bg-[var(--tg-accent)] px-4 py-2 text-sm font-medium text-[var(--tg-accent-ink)] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--tg-accent)]"
            >
              {submitting === 'create' ? 'Creating...' : 'Create Teaching Space'}
            </button>
          </div>
        </form>

        <form
          onSubmit={(e) => void handleAcceptInvite(e)}
          className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs"
        >
          <h2 className="text-lg font-semibold text-[var(--tg-ink)]">Accept an Invitation</h2>
          <p className="mt-1 text-xs text-[var(--tg-muted)]">
            Paste the invitation token shared by your organization administrator.
          </p>

          <div className="mt-4 space-y-4">
            <div>
              <label htmlFor="invite-token" className="block text-sm font-medium text-[var(--tg-ink)]">
                Invitation Token
              </label>
              <input
                id="invite-token"
                type="text"
                required
                value={inviteToken}
                onChange={(e) => setInviteToken(e.target.value)}
                placeholder="Paste invitation token"
                className="mt-1 min-h-11 w-full rounded-lg border border-[var(--tg-border)] px-3 py-2 text-sm text-[var(--tg-ink)] focus-visible:outline-2 focus-visible:outline-[var(--tg-accent)]"
              />
            </div>

            <button
              type="submit"
              disabled={submitting !== null}
              className="inline-flex min-h-11 w-full items-center justify-center rounded-lg border border-[var(--tg-border)] bg-[var(--tg-card)] px-4 py-2 text-sm font-medium text-[var(--tg-ink)] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--tg-accent)]"
            >
              {submitting === 'invite' ? 'Joining...' : 'Join Organization'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
