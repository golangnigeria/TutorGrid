import { useCallback, useEffect, useRef, useState } from "react";
import { Link, Navigate, Route, Routes, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { MotionConfig, motion } from "motion/react";
import { Toaster, toast } from "sonner";
import {
  AuthApiError,
  acceptInvitation,
  fetchMe,
  signOut,
  startOAuthSignIn,
  type MeResponse,
} from "@/lib/auth";
import { Navigation7 } from "@/components/ui/navigation-7";
import { AuthenticatedHeader } from "@/components/AuthenticatedHeader";
import { MobileFooterDock } from "@/components/MobileFooterDock";
import { Hero34 } from "@/components/ui/hero-34";
import { TutorGridLockup } from "@/components/TutorGridLogo";
import { OnboardingView } from "@/components/OnboardingView";
import { WorkspaceView } from "@/components/WorkspaceView";
import "./index.css";

const PENDING_INVITE_STORAGE_KEY = "tg_pending_invite_token";

type SessionState =
  | { status: "loading" }
  | { status: "unauthenticated" }
  | { status: "authenticated"; me: MeResponse }
  | { status: "error"; message: string };

interface SignInPageProps {
  session: SessionState;
}

function SignInPage({ session }: SignInPageProps) {
  const [searchParams] = useSearchParams();
  const [pendingProvider, setPendingProvider] = useState<"google" | "github" | null>(null);
  const [oauthError, setOauthError] = useState<string | null>(() => searchParams.get("error"));
  const nextPath = searchParams.get("next") ?? "/dashboard";

  if (session.status === "authenticated") {
    return <Navigate to={nextPath.startsWith("/") ? nextPath : "/dashboard"} replace />;
  }

  const handleOAuth = async (provider: "google" | "github") => {
    setPendingProvider(provider);
    setOauthError(null);
    try {
      const url = await startOAuthSignIn(provider, nextPath);
      window.location.assign(url);
    } catch (err) {
      setPendingProvider(null);
      setOauthError(
        err instanceof AuthApiError ? err.message : `Unable to start ${provider} sign-in.`,
      );
    }
  };

  return (
    <motion.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 320, damping: 30 }}
      aria-labelledby="sign-in-heading"
      className="mx-auto my-6 max-w-md rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/84 p-7 backdrop-blur-md sm:p-8"
    >
      <div className="flex justify-center">
        <TutorGridLockup />
      </div>

      <h1
        id="sign-in-heading"
        className="mt-4 text-center font-display text-2xl font-semibold tracking-tight text-[#F3F0E8]"
      >
        Sign in to TutorGrid Circle
      </h1>
      <p className="mt-2 text-center text-sm leading-relaxed text-[#A6B0A8]">
        Continue with Google or GitHub to access your teaching spaces, lessons, and live classrooms.
      </p>

      {oauthError && (
        <p
          role="alert"
          className="mt-4 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-3.5 py-2.5 text-sm text-[#F4C2C2]"
        >
          {oauthError}
        </p>
      )}

      <div className="mt-6 flex flex-col gap-3">
        <motion.button
          whileTap={{ scale: 0.985 }}
          type="button"
          disabled={pendingProvider !== null}
          onClick={() => void handleOAuth("google")}
          className="inline-flex min-h-11 w-full cursor-pointer items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2.5 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
        >
          {pendingProvider === "google"
            ? "Redirecting to Google..."
            : "Continue with Google"}
        </motion.button>

        <motion.button
          whileTap={{ scale: 0.985 }}
          type="button"
          disabled={pendingProvider !== null}
          onClick={() => void handleOAuth("github")}
          className="inline-flex min-h-11 w-full cursor-pointer items-center justify-center rounded-xl border border-[#F3F0E8]/18 bg-[#222B24] px-4 py-2.5 text-sm font-medium text-[#F3F0E8] transition-colors hover:border-[#D6C29F]/45 hover:bg-[#2C3329] disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#D6C29F]"
        >
          {pendingProvider === "github"
            ? "Redirecting to GitHub..."
            : "Continue with GitHub"}
        </motion.button>
      </div>
    </motion.section>
  );
}

interface InviteAcceptPageProps {
  session: SessionState;
  onRefreshMe: () => Promise<void>;
}

function InviteAcceptPage({ session, onRefreshMe }: InviteAcceptPageProps) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get("token") ?? "";
  const [pendingProvider, setPendingProvider] = useState<"google" | "github" | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const attemptedRef = useRef(false);

  useEffect(() => {
    if (session.status !== "authenticated" || !token || attemptedRef.current) {
      return;
    }
    attemptedRef.current = true;
    sessionStorage.removeItem(PENDING_INVITE_STORAGE_KEY);
    void (async () => {
      try {
        const res = await acceptInvitation(token);
        await onRefreshMe();
        toast.success(`Joined ${res.membership.tenant_name}`, {
          description: `You are now active as ${res.membership.role}.`,
        });
        navigate("/dashboard", { replace: true });
      } catch (err) {
        setErrorMessage(
          err instanceof AuthApiError ? err.message : "Unable to accept invitation.",
        );
      }
    })();
  }, [session.status, token, onRefreshMe, navigate]);

  if (!token) {
    return (
      <div className="mx-auto my-12 max-w-md rounded-2xl border border-[#D97373]/40 bg-[#181F1A]/86 p-6 text-center backdrop-blur-md">
        <h1 className="font-display text-xl font-semibold text-[#F3F0E8]">
          Invalid Invitation Link
        </h1>
        <p className="mt-2 text-sm text-[#A6B0A8]">
          This invitation link is missing its token. Please open the full link from your email.
        </p>
        <Link
          to="/dashboard"
          className="mt-5 inline-flex min-h-10 items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2 text-sm font-semibold text-[#101411]"
        >
          Go to Dashboard
        </Link>
      </div>
    );
  }

  if (session.status === "unauthenticated") {
    const returnPath = `/invite?token=${encodeURIComponent(token)}`;
    const handleOAuth = async (provider: "google" | "github") => {
      setPendingProvider(provider);
      setErrorMessage(null);
      sessionStorage.setItem(PENDING_INVITE_STORAGE_KEY, token);
      try {
        const url = await startOAuthSignIn(provider, returnPath);
        window.location.assign(url);
      } catch (err) {
        setPendingProvider(null);
        setErrorMessage(
          err instanceof AuthApiError ? err.message : `Unable to start ${provider} sign-in.`,
        );
      }
    };

    return (
      <motion.section
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className="mx-auto my-6 max-w-md rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/86 p-7 backdrop-blur-md sm:p-8"
      >
        <div className="flex justify-center">
          <TutorGridLockup />
        </div>
        <h1 className="mt-4 text-center font-display text-2xl font-semibold text-[#F3F0E8]">
          Accept Your Invitation
        </h1>
        <p className="mt-2 text-center text-sm text-[#A6B0A8]">
          Sign in with the invited email address to join the Teaching Space automatically.
        </p>

        {errorMessage && (
          <p
            role="alert"
            className="mt-4 rounded-xl border border-[#D97373]/40 bg-[#101411]/85 px-3.5 py-2.5 text-sm text-[#F4C2C2]"
          >
            {errorMessage}
          </p>
        )}

        <div className="mt-6 flex flex-col gap-3">
          <button
            type="button"
            disabled={pendingProvider !== null}
            onClick={() => void handleOAuth("google")}
            className="inline-flex min-h-11 w-full cursor-pointer items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2.5 text-sm font-semibold text-[#101411] transition-colors hover:bg-[#A3C2A8] disabled:opacity-60"
          >
            {pendingProvider === "google" ? "Redirecting to Google..." : "Continue with Google"}
          </button>
          <button
            type="button"
            disabled={pendingProvider !== null}
            onClick={() => void handleOAuth("github")}
            className="inline-flex min-h-11 w-full cursor-pointer items-center justify-center rounded-xl border border-[#F3F0E8]/18 bg-[#222B24] px-4 py-2.5 text-sm font-medium text-[#F3F0E8] transition-colors hover:border-[#D6C29F]/45 hover:bg-[#2C3329] disabled:opacity-60"
          >
            {pendingProvider === "github" ? "Redirecting to GitHub..." : "Continue with GitHub"}
          </button>
        </div>
      </motion.section>
    );
  }

  if (errorMessage) {
    return (
      <div className="mx-auto my-12 max-w-md rounded-2xl border border-[#D97373]/40 bg-[#181F1A]/86 p-6 text-center backdrop-blur-md">
        <h1 className="font-display text-xl font-semibold text-[#F3F0E8]">
          Unable to Accept Invitation
        </h1>
        <p role="alert" className="mt-2 text-sm text-[#F4C2C2]">
          {errorMessage}
        </p>
        <div className="mt-5 flex flex-wrap justify-center gap-3">
          <Link
            to="/dashboard"
            className="inline-flex min-h-10 items-center justify-center rounded-xl bg-[#8FAF94] px-4 py-2 text-sm font-semibold text-[#101411]"
          >
            Go to Dashboard
          </Link>
          <Link
            to="/sign-out"
            className="inline-flex min-h-10 items-center justify-center rounded-xl border border-[#F3F0E8]/18 bg-[#222B24] px-4 py-2 text-sm font-medium text-[#F3F0E8]"
          >
            Switch Account
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div
      role="status"
      className="mx-auto my-12 flex max-w-md items-center justify-center gap-3 rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/84 px-6 py-8 text-sm text-[#A6B0A8] backdrop-blur-md"
    >
      <span className="h-2 w-2 animate-pulse rounded-full bg-[#8FAF94]" />
      Joining Teaching Space...
    </div>
  );
}

interface SignOutPageProps {
  onSignedOut: () => void;
}

function SignOutPage({ onSignedOut }: SignOutPageProps) {
  const navigate = useNavigate();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        await signOut();
      } catch (err) {
        if (!cancelled) {
          setErrorMessage(
            err instanceof AuthApiError ? err.message : "Unable to complete sign out.",
          );
          return;
        }
      }
      if (!cancelled) {
        onSignedOut();
        navigate("/sign-in", { replace: true });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [navigate, onSignedOut]);

  return (
    <div className="mx-auto my-12 max-w-md rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/84 p-6 text-center backdrop-blur-md">
      {errorMessage ? (
        <p role="alert" className="text-sm text-[#F4C2C2]">
          {errorMessage}
        </p>
      ) : (
        <div role="status" className="flex items-center justify-center gap-3 text-sm text-[#A6B0A8]">
          <span className="h-2 w-2 animate-pulse rounded-full bg-[#8FAF94]" />
          Signing out of TutorGrid Circle...
        </div>
      )}
    </div>
  );
}

interface DashboardPageProps {
  session: SessionState;
  onRefreshMe: () => Promise<void>;
}

function DashboardPage({ session, onRefreshMe }: DashboardPageProps) {
  if (session.status === "loading") {
    return (
      <div
        role="status"
        className="mx-auto my-12 flex max-w-md items-center justify-center gap-3 rounded-2xl border border-[#F3F0E8]/14 bg-[#181F1A]/82 px-6 py-8 text-sm text-[#A6B0A8] backdrop-blur-md"
      >
        <span className="h-2 w-2 animate-pulse rounded-full bg-[#8FAF94]" />
        Loading your workspace...
      </div>
    );
  }

  if (session.status === "error") {
    return (
      <div
        role="alert"
        className="mx-auto my-12 max-w-md rounded-2xl border border-[#D97373]/40 bg-[#181F1A]/88 p-4 text-sm text-[#F4C2C2] backdrop-blur-md"
      >
        {session.message}
      </div>
    );
  }

  if (session.status === "unauthenticated") {
    return <Navigate to="/sign-in" replace />;
  }

  const storedInvite = sessionStorage.getItem(PENDING_INVITE_STORAGE_KEY);
  if (storedInvite) {
    sessionStorage.removeItem(PENDING_INVITE_STORAGE_KEY);
    return <Navigate to={`/invite?token=${encodeURIComponent(storedInvite)}`} replace />;
  }

  if (session.me.memberships.length === 0) {
    return <OnboardingView user={session.me.user} onCompleted={onRefreshMe} />;
  }

  return <WorkspaceView me={session.me} onRefreshMe={onRefreshMe} />;
}

function HomeRoute({ isAuthenticated }: { isAuthenticated: boolean }) {
  const [searchParams] = useSearchParams();
  const legacyInvite = searchParams.get("invite");
  if (legacyInvite) {
    return <Navigate to={`/invite?token=${encodeURIComponent(legacyInvite)}`} replace />;
  }
  return <Hero34 isAuthenticated={isAuthenticated} />;
}

function App() {
  const [session, setSession] = useState<SessionState>({ status: "loading" });
  const location = useLocation();

  const loadSession = useCallback(async () => {
    setSession({ status: "loading" });
    try {
      const me = await fetchMe();
      setSession({ status: "authenticated", me });
    } catch (err) {
      if (err instanceof AuthApiError && err.status === 401) {
        setSession({ status: "unauthenticated" });
        return;
      }
      setSession({
        status: "error",
        message:
          err instanceof AuthApiError ? err.message : "Unable to reach TutorGrid Circle API.",
      });
    }
  }, []);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  useEffect(() => {
    if (location.hash) {
      const el = document.getElementById(location.hash.slice(1));
      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }
  }, [location.pathname, location.hash]);

  const handleSignedOut = useCallback(() => {
    setSession({ status: "unauthenticated" });
  }, []);

  return (
    <MotionConfig reducedMotion="user">
      <Toaster
        theme="dark"
        position="top-center"
        toastOptions={{
          style: {
            background: "#181F1A",
            border: "1px solid rgba(243, 240, 232, 0.14)",
            color: "#F3F0E8",
          },
        }}
      />

      <div className="relative flex min-h-screen flex-col overflow-x-hidden bg-[#101411] text-[#F3F0E8]">
        {/* Full-bleed visible dusk landscape backdrop with light pine-charcoal tint */}
        <div aria-hidden="true" className="pointer-events-none fixed inset-0 z-0">
          <img
            src="/tutorgrid-bg.avif"
            alt=""
            className="h-full w-full object-cover object-center opacity-80"
          />
          <div className="absolute inset-0 bg-[#101411]/38" />
        </div>

        {session.status === "authenticated" ? (
          <AuthenticatedHeader me={session.me} onRefreshMe={loadSession} />
        ) : (
          <Navigation7 />
        )}

        <main className="relative z-10 mx-auto w-full max-w-5xl flex-1 px-4 pt-24 pb-28 sm:px-6">
          <Routes>
            <Route
              path="/"
              element={<HomeRoute isAuthenticated={session.status === "authenticated"} />}
            />
            <Route path="/sign-in" element={<SignInPage session={session} />} />
            <Route path="/sign-out" element={<SignOutPage onSignedOut={handleSignedOut} />} />
            <Route
              path="/invite"
              element={<InviteAcceptPage session={session} onRefreshMe={loadSession} />}
            />
            <Route
              path="/dashboard"
              element={<DashboardPage session={session} onRefreshMe={loadSession} />}
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>

        <MobileFooterDock isAuthenticated={session.status === "authenticated"} />
      </div>
    </MotionConfig>
  );
}

export default App;
