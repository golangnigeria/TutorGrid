import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';

describe('App auth and onboarding shell', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows Google and GitHub sign-in buttons when unauthenticated (401)', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'UNAUTHORIZED', message: 'authentication required' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Sign in to TutorGrid/i })).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /Continue with Google/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Continue with GitHub/i })).toBeInTheDocument();
    expect(screen.queryByLabelText(/password/i)).not.toBeInTheDocument();
  });

  it('shows onboarding screen when authenticated with zero memberships', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user: { id: 'u-1', display_name: 'Ada Lovelace', email: 'ada@example.com' },
          memberships: [],
          active_tenant_id: null,
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Welcome, Ada Lovelace/i })).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /Create Teaching Space/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Join Organization/i })).toBeInTheDocument();
  });

  it('shows active workspace and members when authenticated with membership', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            user: { id: 'u-1', display_name: 'Ada Lovelace', email: 'ada@example.com' },
            memberships: [
              { tenant_id: 't-1', tenant_name: 'Math Studio', tenant_slug: 'math-studio', role: 'org_admin' },
            ],
            active_tenant_id: 't-1',
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            members: [
              {
                membership_id: 'm-1',
                user_id: 'u-1',
                display_name: 'Ada Lovelace',
                email: 'ada@example.com',
                role: 'org_admin',
              },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Math Studio/i })).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /Create Invitation/i })).toBeInTheDocument();
  });
});
