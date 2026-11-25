import { useState } from 'react';
import { ApiError, startOAuthSignIn } from '../../lib/api';

interface SignInViewProps {
  oauthError: string | null;
}

export function SignInView({ oauthError }: SignInViewProps) {
  const [pendingProvider, setPendingProvider] = useState<'google' | 'github' | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(oauthError);

  const handleOAuth = async (provider: 'google' | 'github') => {
    setPendingProvider(provider);
    setErrorMessage(null);
    try {
      const url = await startOAuthSignIn(provider);
      window.location.assign(url);
    } catch (err) {
      setPendingProvider(null);
      if (err instanceof ApiError) {
        setErrorMessage(err.message);
      } else {
        setErrorMessage(`Unable to start ${provider} sign-in.`);
      }
    }
  };

  return (
    <section
      aria-labelledby="sign-in-heading"
      className="mx-auto max-w-md rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs sm:p-8"
    >
      <h1 id="sign-in-heading" className="text-xl font-semibold text-[var(--tg-ink)]">
        Sign in to TutorGrid
      </h1>
      <p className="mt-2 text-sm text-[var(--tg-muted)]">
        Use your Google or GitHub account to access Lesson Studio, Whiteboard, Activities, and Live Classroom.
      </p>

      {errorMessage && (
        <p
          role="alert"
          className="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-[var(--tg-error)]"
        >
          {errorMessage}
        </p>
      )}

      <div className="mt-6 flex flex-col gap-3">
        <button
          type="button"
          disabled={pendingProvider !== null}
          onClick={() => void handleOAuth('google')}
          className="inline-flex min-h-11 w-full items-center justify-center rounded-lg bg-[var(--tg-accent)] px-4 py-2.5 text-sm font-medium text-[var(--tg-accent-ink)] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--tg-accent)]"
        >
          {pendingProvider === 'google' ? 'Redirecting to Google...' : 'Continue with Google'}
        </button>

        <button
          type="button"
          disabled={pendingProvider !== null}
          onClick={() => void handleOAuth('github')}
          className="inline-flex min-h-11 w-full items-center justify-center rounded-lg border border-[var(--tg-border)] bg-[var(--tg-card)] px-4 py-2.5 text-sm font-medium text-[var(--tg-ink)] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--tg-accent)]"
        >
          {pendingProvider === 'github' ? 'Redirecting to GitHub...' : 'Continue with GitHub'}
        </button>
      </div>
    </section>
  );
}
