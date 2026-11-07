// @vitest-environment jsdom
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

interface GoogleInitConfig {
  client_id: string;
  ux_mode?: string;
  use_fedcm_for_button?: boolean;
  callback: (response: { credential?: string }) => void;
}

interface GoogleButtonConfig {
  text?: string;
  theme?: string;
  width?: number;
}

type GoogleId = NonNullable<NonNullable<Window["google"]>["accounts"]>["id"];

let initConfig: GoogleInitConfig | null = null;
let buttonConfig: GoogleButtonConfig | null = null;
let buttonParent: HTMLElement | null = null;
let renderButtonMock: ReturnType<typeof vi.fn>;
// The deprecated One Tap entry point. The component's types no longer declare
// it, so the mock is only here to prove the component never calls it.
let promptMock: ReturnType<typeof vi.fn>;

function installGoogle(): void {
  initConfig = null;
  buttonConfig = null;
  buttonParent = null;
  promptMock = vi.fn();
  renderButtonMock = vi.fn((parent: HTMLElement, options: GoogleButtonConfig) => {
    buttonParent = parent;
    buttonConfig = options;
  });
  window.google = {
    accounts: {
      id: {
        initialize: vi.fn((config: GoogleInitConfig) => {
          initConfig = config;
        }),
        renderButton: renderButtonMock,
        prompt: promptMock,
      } as unknown as GoogleId,
    },
  };
}

async function renderGoogleButton(
  clientId: string,
  onCredential: (token: string) => Promise<void>,
  text?: "signin_with" | "signup_with" | "continue_with" | "signin"
) {
  vi.resetModules();
  vi.doMock("@/lib/config", () => ({ GOOGLE_CLIENT_ID: clientId }));
  // Imported together, after resetModules, so the button and the provider share
  // one instance of the toast context module.
  const [{ GoogleSignInButton }, { ToastProvider }] = await Promise.all([
    import("@/components/google-signin"),
    import("@/components/toast"),
  ]);
  return render(
    <ToastProvider>
      <GoogleSignInButton text={text} onCredential={onCredential} />
    </ToastProvider>
  );
}

afterEach(() => {
  cleanup();
  vi.doUnmock("@/lib/config");
  vi.resetModules();
  delete window.google;
  initConfig = null;
  buttonConfig = null;
  buttonParent = null;
});

describe("GoogleSignInButton", () => {
  it("initialises Google and renders Google's own button", async () => {
    installGoogle();
    await renderGoogleButton("test-client.apps.googleusercontent.com", vi.fn());

    await waitFor(() => expect(renderButtonMock).toHaveBeenCalledTimes(1));
    expect(initConfig?.client_id).toBe("test-client.apps.googleusercontent.com");
    expect(initConfig?.ux_mode).toBe("popup");
    expect(initConfig?.use_fedcm_for_button).toBe(true);
    expect(buttonConfig?.text).toBe("continue_with");
    expect(buttonParent).not.toBeNull();
    expect(document.body.contains(buttonParent)).toBe(true);
    // Google draws the button; CampVote renders no button of its own.
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("explains a missing client ID instead of falling back", async () => {
    installGoogle();
    await renderGoogleButton("", vi.fn());

    expect(screen.getByRole("alert").textContent).toMatch(/Google sign-in is unavailable/i);
    expect(renderButtonMock).not.toHaveBeenCalled();
    expect(promptMock).not.toHaveBeenCalled();
  });

  it("passes the requested label through to Google's button", async () => {
    installGoogle();
    await renderGoogleButton("client", vi.fn(), "signup_with");

    await waitFor(() => expect(renderButtonMock).toHaveBeenCalledTimes(1));
    expect(buttonConfig?.text).toBe("signup_with");
  });

  it("never calls the deprecated One Tap prompt", async () => {
    installGoogle();
    await renderGoogleButton("client", vi.fn());

    await waitFor(() => expect(renderButtonMock).toHaveBeenCalledTimes(1));
    expect(promptMock).not.toHaveBeenCalled();
  });

  it("sends a credential to the backend and shows a loading state", async () => {
    installGoogle();
    let resolveBackend: () => void = () => undefined;
    const onCredential = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveBackend = resolve;
        })
    );
    await renderGoogleButton("client", onCredential);
    await waitFor(() => expect(initConfig).not.toBeNull());

    await act(async () => {
      initConfig?.callback({ credential: "google-id-token" });
    });

    expect(onCredential).toHaveBeenCalledWith("google-id-token");
    expect(screen.getByRole("status").textContent).toMatch(/Verifying with CampVote/i);

    await act(async () => resolveBackend());
    await waitFor(() => expect(screen.queryByRole("status")).toBeNull());
  });

  it("does not submit twice while the first request is in flight", async () => {
    installGoogle();
    let resolveBackend: () => void = () => undefined;
    const onCredential = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveBackend = resolve;
        })
    );
    await renderGoogleButton("client", onCredential);
    await waitFor(() => expect(initConfig).not.toBeNull());

    await act(async () => {
      initConfig?.callback({ credential: "token-1" });
      initConfig?.callback({ credential: "token-2" });
    });

    expect(onCredential).toHaveBeenCalledTimes(1);
    await act(async () => resolveBackend());
  });

  it("shows the student-domain refusal as a visible toast", async () => {
    installGoogle();
    const { ApiError } = await import("@/lib/api");
    const onCredential = vi.fn().mockRejectedValue(
      new ApiError(
        "Only @std.must.ac.ug student or @must.ac.ug staff accounts can sign in with Google.",
        403
      )
    );
    await renderGoogleButton("client", onCredential);
    await waitFor(() => expect(initConfig).not.toBeNull());

    await act(async () => {
      initConfig?.callback({ credential: "token" });
    });

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toMatch(/@std\.must\.ac\.ug/);
    // The message sits in the notification region, not inline under the button.
    expect(alert.closest(".toasts")).not.toBeNull();
    expect(document.querySelector(".err")).toBeNull();
  });

  it("shows a network failure without leaking internals", async () => {
    installGoogle();
    const onCredential = vi.fn().mockRejectedValue(new Error("socket hang up"));
    await renderGoogleButton("client", onCredential);
    await waitFor(() => expect(initConfig).not.toBeNull());

    await act(async () => {
      initConfig?.callback({ credential: "token" });
    });

    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/Check your connection and try again/i);
    });
    expect(screen.getByRole("alert").textContent).not.toMatch(/socket hang up/i);
  });

  it("handles a Google response that carries no credential", async () => {
    installGoogle();
    const onCredential = vi.fn();
    await renderGoogleButton("client", onCredential);
    await waitFor(() => expect(initConfig).not.toBeNull());

    await act(async () => {
      initConfig?.callback({});
    });

    expect(onCredential).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toMatch(/did not return a sign-in credential/i);
  });
});
