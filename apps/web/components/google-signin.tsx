"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api";
import { GOOGLE_CLIENT_ID } from "@/lib/config";
import { useToast } from "./toast";

interface GoogleCredentialResponse {
  credential?: string;
}

type GoogleButtonText = "signin_with" | "signup_with" | "continue_with" | "signin";

/**
 * The subset of Google's GsiButtonConfiguration this button sets. Google's SDK
 * draws the button itself, so these are the only choices CampVote makes; the
 * size, colours and brand rules stay Google's.
 */
interface GoogleButtonConfig {
  type?: "standard" | "icon";
  theme?: "outline" | "filled_blue" | "filled_black";
  size?: "large" | "medium" | "small";
  text?: GoogleButtonText;
  shape?: "rectangular" | "pill" | "circle" | "square";
  logo_alignment?: "left" | "center";
  width?: number;
}

interface GoogleIdConfig {
  client_id: string;
  callback: (response: GoogleCredentialResponse) => void;
  ux_mode?: "popup" | "redirect";
  auto_select?: boolean;
  /**
   * Runs the button flow through FedCM. The button flow already works with
   * third-party cookies blocked; FedCM is how Google keeps the personalised
   * button and removes the old One Tap cookie dependency, so this states the
   * intent rather than relying on the default.
   */
  use_fedcm_for_button?: boolean;
}

interface GoogleAccountsId {
  initialize: (config: GoogleIdConfig) => void;
  renderButton: (parent: HTMLElement, options: GoogleButtonConfig) => void;
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
 * Google's button cannot be restyled, so its theme follows the page's. The
 * "outline" treatment is light; a dark page gets the black button instead of a
 * white block sitting on it.
 */
function googleButtonTheme(): NonNullable<GoogleButtonConfig["theme"]> {
  if (typeof document === "undefined") return "outline";
  const explicit = document.documentElement.getAttribute("data-theme");
  if (explicit === "dark") return "filled_black";
  if (explicit === "light") return "outline";
  return typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-color-scheme: dark)").matches
    ? "filled_black"
    : "outline";
}

interface GoogleSignInButtonProps {
  /**
   * Receives the raw ID token. It must send it to the backend and must not
   * decode it or treat its claims as trusted identity data.
   */
  onCredential: (idToken: string) => Promise<void>;
  text?: GoogleButtonText;
}

/**
 * The "Continue with Google" option, drawn by Google Identity Services.
 *
 * Signing in is a click on Google's own button (`renderButton`), which runs the
 * button flow: it does not depend on One Tap or on third-party cookies, and it
 * never touches the One Tap prompt-status methods that FedCM deprecates. An
 * earlier version called `google.accounts.id.prompt()` and translated its
 * suppression into "allow third-party cookies" — which is both a dead end for
 * the user and the cause of the GSI FedCM warning.
 *
 * A credential from Google is not a CampVote sign-in: it is handed to
 * onCredential, which resolves only after the backend has verified it.
 */
export function GoogleSignInButton({
  onCredential,
  text = "continue_with",
}: GoogleSignInButtonProps) {
  const callbackRef = useRef(onCredential);
  const busyRef = useRef(false);
  const initializedRef = useRef(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const toast = useToast();

  const [busy, setBusy] = useState(false);

  useEffect(() => {
    callbackRef.current = onCredential;
  }, [onCredential]);

  const submit = useCallback(
    async (credential: string) => {
      // A repeated callback must not open two sessions.
      if (busyRef.current) return;
      busyRef.current = true;
      setBusy(true);
      try {
        await callbackRef.current(credential);
      } catch (err) {
        // A toast, not inline text: the API's refusal (for example "Only
        // @std.must.ac.ug student or @must.ac.ug staff accounts can sign in
        // with Google.") is the whole answer to the user's action and must be
        // impossible to miss.
        toast.error(googleErrorMessage(err));
      } finally {
        busyRef.current = false;
        setBusy(false);
      }
    },
    [toast]
  );

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) return;
    let cancelled = false;

    // Google draws its button into this element. Running it again after a theme
    // change replaces the old button rather than stacking a second one.
    const render = () => {
      const googleId = window.google?.accounts?.id;
      const container = containerRef.current;
      if (!googleId || !container) return;
      container.replaceChildren();
      googleId.renderButton(container, {
        type: "standard",
        theme: googleButtonTheme(),
        size: "large",
        text,
        shape: "rectangular",
        logo_alignment: "left",
        // The auth card's inner width (440px max, 28px padding each side).
        width: 384,
      });
    };

    loadGoogleScript()
      .then(() => {
        if (cancelled) return;
        const googleId = window.google?.accounts?.id;
        if (!googleId || !containerRef.current) {
          toast.error("Google sign-in could not start. Check your connection and try again.");
          return;
        }
        if (!initializedRef.current) {
          initializedRef.current = true;
          googleId.initialize({
            client_id: GOOGLE_CLIENT_ID,
            ux_mode: "popup",
            auto_select: false,
            use_fedcm_for_button: true,
            callback: (response) => {
              if (!response.credential) {
                toast.error("Google did not return a sign-in credential. Please try again.");
                return;
              }
              void submit(response.credential);
            },
          });
        }
        render();
      })
      .catch(() => {
        if (!cancelled) {
          toast.error("Google sign-in could not load. Check your connection and try again.");
        }
      });

    // A dark/light switch has to re-render Google's button to re-theme it.
    const observer =
      typeof MutationObserver === "undefined" ? null : new MutationObserver(() => render());
    observer?.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-theme"],
    });

    return () => {
      cancelled = true;
      observer?.disconnect();
    };
  }, [submit, text, toast]);

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
      <div
        ref={containerRef}
        aria-busy={busy}
        style={{
          display: "flex",
          justifyContent: "center",
          minHeight: 44,
          opacity: busy ? 0.5 : 1,
          pointerEvents: busy ? "none" : "auto",
        }}
      />
      {busy ? (
        <span className="hint" role="status">
          Verifying with CampVote…
        </span>
      ) : null}
    </div>
  );
}
