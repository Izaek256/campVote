"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api";
import { GOOGLE_CLIENT_ID } from "@/lib/config";

interface GoogleCredentialResponse {
  credential?: string;
}

/**
 * The subset of Google's MomentNotification this button reacts to. One Tap can
 * be suppressed (third-party cookies blocked, origin not registered), and the
 * reason is the only thing that can explain a click that appears to do nothing.
 */
interface GoogleMomentNotification {
  isNotDisplayed?: () => boolean;
  isSkippedMoment?: () => boolean;
  getNotDisplayedReason?: () => string;
  getSkippedReason?: () => string;
}

interface GoogleIdConfig {
  client_id: string;
  callback: (response: GoogleCredentialResponse) => void;
  ux_mode?: "popup" | "redirect";
  auto_select?: boolean;
  cancel_on_tap_outside?: boolean;
}

interface GoogleAccountsId {
  initialize: (config: GoogleIdConfig) => void;
  prompt: (momentListener?: (notification: GoogleMomentNotification) => void) => void;
}

declare global {
  interface Window {
    google?: {
      accounts?: {
        id?: GoogleAccountsId;
      };
    };
  }
}

/**
 * Load Google Identity Services once per page, even when the component mounts
 * twice in development or the user navigates between sign-in pages.
 *
 * The script is appended after hydration, so the server and the first client
 * render agree and no third-party script blocks the page.
 */
let scriptPromise: Promise<void> | null = null;

function loadGoogleScript(): Promise<void> {
  if (typeof window === "undefined") return Promise.resolve();
  if (window.google?.accounts?.id) return Promise.resolve();
  if (scriptPromise) return scriptPromise;

  scriptPromise = new Promise<void>((resolve, reject) => {
    const fail = () => {
      scriptPromise = null;
      reject(new Error("Google sign-in script failed to load"));
    };
    const existing = document.querySelector<HTMLScriptElement>(
      'script[data-campvote-google-signin], script[src="https://accounts.google.com/gsi/client"]'
    );
    if (existing) {
      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener("error", fail, { once: true });
      return;
    }

    const script = document.createElement("script");
    script.src = "https://accounts.google.com/gsi/client";
    script.async = true;
    script.defer = true;
    script.dataset.campvoteGoogleSignin = "true";
    script.addEventListener("load", () => resolve(), { once: true });
    script.addEventListener("error", fail, { once: true });
    document.head.appendChild(script);
  });
  return scriptPromise;
}

/**
 * The API's refusal is written for a person, so show it. Anything else is a
 * transport or browser failure and gets one safe sentence: a raw exception
 * message can name hosts and internals.
 *
 * The shape check as well as instanceof covers an ApiError that crossed a
 * module boundary (tests, an iframe) and lost its prototype.
 */
function googleErrorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  if (
    error !== null &&
    typeof error === "object" &&
    typeof (error as { status?: unknown }).status === "number" &&
    typeof (error as { message?: unknown }).message === "string"
  ) {
    return (error as { message: string }).message;
  }
  return "Google sign-in failed. Check your connection and try again.";
}

/**
 * Turn One Tap's suppression reason into one sentence a person can act on.
 * A dismissed moment needs no message: the user closed it on purpose.
 */
function oneTapMessage(notification: GoogleMomentNotification): string {
  if (notification.isNotDisplayed?.()) {
    switch (notification.getNotDisplayedReason?.()) {
      case "browser_not_supported":
        return "This browser cannot show Google sign-in. Try another browser.";
      case "unregistered_origin":
        return "This site is not registered with the Google client. An administrator must add this origin in Google Cloud Console.";
      case "invalid_client":
        return "Google rejected this client ID. An administrator must check GOOGLE_OAUTH_CLIENT_ID and NEXT_PUBLIC_GOOGLE_CLIENT_ID.";
      default:
        return "Google sign-in could not open. Allow third-party cookies for this site, then try again.";
    }
  }
  if (notification.isSkippedMoment?.()) {
    return "Google sign-in was skipped. Please try again.";
  }
  return "";
}

/**
 * The official Google "G" mark in Google's four brand colours. It is inlined
 * rather than themed so the brand colours are exact.
 */
function GoogleMark() {
  return (
    <svg width="18" height="18" viewBox="0 0 48 48" aria-hidden="true" focusable="false">
      <path
        fill="#EA4335"
        d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
      />
      <path
        fill="#4285F4"
        d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
      />
      <path
        fill="#FBBC05"
        d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
      />
      <path
        fill="#34A853"
        d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
      />
    </svg>
  );
}

const LABELS: Record<NonNullable<GoogleButtonText>, string> = {
  signin_with: "Sign in with Google",
  signup_with: "Sign up with Google",
  continue_with: "Continue with Google",
  signin: "Sign in with Google",
};

type GoogleButtonText = "signin_with" | "signup_with" | "continue_with" | "signin";

interface GoogleSignInButtonProps {
  /**
   * Receives the raw ID token. It must send it to the backend and must not
   * decode it or treat its claims as trusted identity data.
   */
  onCredential: (idToken: string) => Promise<void>;
  text?: GoogleButtonText;
}

/**
 * The "Continue with Google" option.
 *
 * The button is CampVote's own so it matches the rest of the interface, but the
 * credential comes from Google: clicking calls Google's One Tap, and only the
 * credential it returns is used. A successful One Tap is not a CampVote
 * sign-in either — the credential is handed to onCredential, which only
 * resolves after the backend has verified it.
 */
export function GoogleSignInButton({
  onCredential,
  text = "continue_with",
}: GoogleSignInButtonProps) {
  const callbackRef = useRef(onCredential);
  const busyRef = useRef(false);
  const initializedRef = useRef(false);

  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    callbackRef.current = onCredential;
  }, [onCredential]);

  const submit = useCallback(async (credential: string) => {
    // A repeated One Tap callback must not open two sessions.
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError("");
    try {
      await callbackRef.current(credential);
    } catch (err) {
      setError(googleErrorMessage(err));
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) return;
    let cancelled = false;

    loadGoogleScript()
      .then(() => {
        if (cancelled || initializedRef.current) return;
        const googleId = window.google?.accounts?.id;
        if (!googleId) {
          setError("Google sign-in could not start. Check your connection and try again.");
          return;
        }
        initializedRef.current = true;
        googleId.initialize({
          client_id: GOOGLE_CLIENT_ID,
          ux_mode: "popup",
          auto_select: false,
          cancel_on_tap_outside: true,
          callback: (response) => {
            if (!response.credential) {
              setError("Google did not return a sign-in credential. Please try again.");
              return;
            }
            void submit(response.credential);
          },
        });
        setReady(true);
      })
      .catch(() => {
        if (!cancelled) {
          setError("Google sign-in could not load. Check your connection and try again.");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [submit]);

  const startGoogleSignIn = useCallback(() => {
    const googleId = window.google?.accounts?.id;
    if (!googleId?.prompt) {
      setError("Google sign-in is not ready yet. Please try again in a moment.");
      return;
    }
    setError("");
    googleId.prompt((notification) => {
      const message = oneTapMessage(notification);
      if (message) setError(message);
    });
  }, []);

  if (!GOOGLE_CLIENT_ID) {
    return (
      <div className="banner" role="alert">
        <div>
          <b>Google sign-in is unavailable.</b> This deployment has no Google client configured.
          Please contact the election administrator.
        </div>
      </div>
    );
  }

  return (
    <div className="form-stack" style={{ gap: 8 }}>
      <button
        type="button"
        className="btn block lg"
        onClick={startGoogleSignIn}
        disabled={!ready || busy}
      >
        <GoogleMark />
        {busy ? "Verifying…" : LABELS[text]}
      </button>
      {busy ? (
        <span className="hint" role="status">
          Verifying with CampVote…
        </span>
      ) : null}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
