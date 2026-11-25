import { useCallback, useEffect, useState } from 'react';
import { ApiError, fetchMe, signOut, type MeResponse } from './lib/api';
import { SignInView } from './features/auth/SignInView';
import { OnboardingView } from './features/auth/OnboardingView';
import { WorkspaceView } from './features/auth/WorkspaceView';

type AuthState =
  | { status: 'loading' }
  | { status: 'unauthenticated' }
  | { status: 'authenticated'; me: MeResponse }
  | { status: 'error'; message: string };

export function App() {
  const [auth, setAuth] = useState<AuthState>({ status: 'loading' });
  const [signingOut, setSigningOut] = useState(false);

  const oauthError = new URLSearchParams(window.location.search).get('error');

  const loadSession = useCallback(async () => {
    setAuth({ status: 'loading' });
    try {
      const me = await fetchMe();
      setAuth({ status: 'authenticated', me });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setAuth({ status: 'unauthenticated' });
        return;
      }
      setAuth({
        status: 'error',
        message: err instanceof ApiError ? err.message : 'Unable to reach TutorGrid API.',
      });
    }
  }, []);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  const handleSignOut = async () => {
    setSigningOut(true);
    try {
      await signOut();
      setAuth({ status: 'unauthenticated' });
    } finally {
      setSigningOut(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--tg-border)] bg-[var(--tg-card)]">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4 sm:px-6">
          <div>
            <p className="text-lg font-semibold tracking-tight text-[var(--tg-ink)]">TutorGrid</p>
            <p className="text-xs text-[var(--tg-muted)]">
              Lesson Studio · Whiteboard · Activities · Live Classroom
            </p>
          </div>

          {auth.status === 'authenticated' && (
            <button
              type="button"
              disabled={signingOut}
              onClick={() => void handleSignOut()}
              className="inline-flex min-h-11 items-center justify-center rounded-lg border border-[var(--tg-border)] px-3 py-2 text-xs font-medium text-[var(--tg-ink)] disabled:opacity-60"
            >
              {signingOut ? 'Signing out...' : 'Sign out'}
            </button>
          )}
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 sm:px-6">
        {auth.status === 'loading' && (
          <p role="status" className="text-sm text-[var(--tg-muted)]">
            Checking session...
          </p>
        )}

        {auth.status === 'error' && (
          <div role="alert" className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-[var(--tg-error)]">
            {auth.message}
          </div>
        )}

        {auth.status === 'unauthenticated' && <SignInView oauthError={oauthError} />}

        {auth.status === 'authenticated' && auth.me.memberships.length === 0 && (
          <OnboardingView user={auth.me.user} onCompleted={loadSession} />
        )}

        {auth.status === 'authenticated' && auth.me.memberships.length > 0 && (
          <WorkspaceView me={auth.me} onRefreshMe={loadSession} />
        )}
      </main>
    </div>
  );
}
