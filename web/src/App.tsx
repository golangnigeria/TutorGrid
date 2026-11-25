import { useCallback, useEffect, useState } from 'react';

type ReadinessState =
  | { status: 'loading' }
  | { status: 'ready'; database: string }
  | { status: 'error'; message: string };

interface ReadyResponse {
  status?: string;
  database?: string;
}

export function App() {
  const [readiness, setReadiness] = useState<ReadinessState>({ status: 'loading' });

  const checkReadiness = useCallback(async () => {
    setReadiness({ status: 'loading' });
    try {
      const response = await fetch('/readyz', {
        headers: { Accept: 'application/json' },
      });
      const payload = (await response.json()) as ReadyResponse;
      if (!response.ok || payload.status !== 'ready') {
        setReadiness({
          status: 'error',
          message: `API readiness check failed (HTTP ${response.status})`,
        });
        return;
      }
      setReadiness({
        status: 'ready',
        database: payload.database ?? 'ok',
      });
    } catch {
      setReadiness({
        status: 'error',
        message: 'Unable to reach TutorGrid API (/readyz).',
      });
    }
  }, []);

  useEffect(() => {
    void checkReadiness();
  }, [checkReadiness]);

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--tg-border)] bg-[var(--tg-card)]">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4 sm:px-6">
          <div>
            <p className="text-lg font-semibold tracking-tight text-[var(--tg-ink)]">
              TutorGrid
            </p>
            <p className="text-xs text-[var(--tg-muted)]">
              Lesson Studio · Whiteboard · Activities · Live Classroom
            </p>
          </div>
          <span className="rounded-full border border-[var(--tg-border)] px-3 py-1 text-xs font-medium text-[var(--tg-muted)]">
            Foundation Shell
          </span>
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 sm:px-6">
        <section
          aria-labelledby="system-status-heading"
          className="rounded-xl border border-[var(--tg-border)] bg-[var(--tg-card)] p-6 shadow-xs"
        >
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1
                id="system-status-heading"
                className="text-xl font-semibold text-[var(--tg-ink)]"
              >
                Platform Readiness
              </h1>
              <p className="mt-1 text-sm text-[var(--tg-muted)]">
                Unauthenticated operational health check for the TutorGrid Go API and PostgreSQL.
              </p>
            </div>
            <button
              type="button"
              onClick={() => void checkReadiness()}
              className="inline-flex min-h-11 items-center justify-center rounded-lg bg-[var(--tg-accent)] px-4 py-2 text-sm font-medium text-[var(--tg-accent-ink)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--tg-accent)]"
            >
              Recheck Readiness
            </button>
          </div>

          <div className="mt-6 rounded-lg border border-[var(--tg-border)] bg-[var(--tg-surface)] p-4">
            {readiness.status === 'loading' && (
              <p role="status" className="text-sm text-[var(--tg-muted)]">
                Checking API and database readiness...
              </p>
            )}

            {readiness.status === 'ready' && (
              <p role="status" className="text-sm font-medium text-[var(--tg-success)]">
                API Ready — PostgreSQL status: {readiness.database}
              </p>
            )}

            {readiness.status === 'error' && (
              <p role="alert" className="text-sm font-medium text-[var(--tg-error)]">
                {readiness.message}
              </p>
            )}
          </div>
        </section>
      </main>
    </div>
  );
}
