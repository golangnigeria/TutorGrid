import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';

describe('App readiness shell', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('displays ready status when /readyz returns 200 ready', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ status: 'ready', database: 'ok' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    render(<App />);

    expect(screen.getByRole('status')).toHaveTextContent(/Checking API and database readiness/i);

    await waitFor(() => {
      expect(screen.getByRole('status')).toHaveTextContent(/API Ready — PostgreSQL status: ok/i);
    });
  });

  it('displays error alert when /readyz returns 503', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ status: 'not_ready', database: 'unreachable' }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/API readiness check failed \(HTTP 503\)/i);
    });
  });
});
