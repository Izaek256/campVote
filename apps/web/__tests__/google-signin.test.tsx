// @vitest-environment jsdom
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

interface GoogleInitConfig {
  client_id: string;
  ux_mode?: string;
  callback: (response: { credential?: string }) => void;
}

interface GoogleMomentNotification {
  isNotDisplayed?: () => boolean;
  isSkippedMoment?: () => boolean;
  getNotDisplayedReason?: () => string;
  getSkippedReason?: () => string;
}

type GoogleId = NonNullable<NonNullable<Window["google"]>["accounts"]>["id"];

let initConfig: GoogleInitConfig | null = null;
let promptMock: ReturnType<typeof vi.fn>;
let momentListener: ((notification: GoogleMomentNotification) => void) | null = null;

function installGoogle(moment?: GoogleMomentNotification): void {
  initConfig = null;
  momentListener = null;
  promptMock = vi.fn((listener?: (notification: GoogleMomentNotification) => void) => {
    momentListener = listener ?? null;
    if (moment && listener) listener(moment);
  });
  window.google = {
    accounts: {
      id: {
        initialize: vi.fn((config: GoogleInitConfig) => {
          initConfig = config;
        }) as unknown as NonNullable<GoogleId>["initialize"],
        prompt: promptMock as unknown as NonNullable<GoogleId>["prompt"],
      },
    },
  };
}

async function renderGoogleButton(clientId: string, onCredential: (token: string) => Promise<void>) {
  vi.resetModules();
  vi.doMock("@/lib/config", () => ({ GOOGLE_CLIENT_ID: clientId }));
  const { GoogleSignInButton } = await import("@/components/google-signin");
  return render(<GoogleSignInButton onCredential={onCredential} />);
}

// The repository does not install jest-dom matchers, so assert the DOM
// property directly instead of using toBeEnabled().
function buttonDisabled(): boolean {
  return (screen.getByRole("button") as HTMLButtonElement).disabled;
}

afterEach(() => {
  cleanup();
  vi.doUnmock("@/lib/config");
  vi.resetModules();
  delete window.google;
  initConfig = null;
  momentListener = null;
});

describe("GoogleSignInButton", () => {
  it("initialises Google with the configured client and shows our own button", async () => {
    installGoogle();
    await renderGoogleButton("test-client.apps.googleusercontent.com", vi.fn());

    await waitFor(() => expect(initConfig).not.toBeNull());
    expect(initConfig?.client_id).toBe("test-client.apps.googleusercontent.com");
    expect(initConfig?.ux_mode).toBe("popup");

    await waitFor(() => expect(buttonDisabled()).toBe(false));
    expect(screen.getByRole("button").textContent).toMatch(/Continue with Google/);
  });

  it("explains a missing client ID instead of falling back", async () => {
    installGoogle();
    await renderGoogleButton("", vi.fn());

    expect(screen.getByRole("alert").textContent).toMatch(/Google sign-in is unavailable/i);
    expect(screen.queryByRole("button")).toBeNull();
    expect(promptMock).not.toHaveBeenCalled();
  });

  it("calls Google One Tap when the button is clicked", async () => {
    installGoogle();
    await renderGoogleButton("client", vi.fn());
    await waitFor(() => expect(buttonDisabled()).toBe(false));

    await act(async () => {
      screen.getByRole("button").click();
    });

    expect(promptMock).toHaveBeenCalledTimes(1);
  });

  it("explains a suppressed One Tap instead of failing silently", async () => {
    installGoogle({
      isNotDisplayed: () => true,
      getNotDisplayedReason: () => "unregistered_origin",
    });
    await renderGoogleButton("client", vi.fn());
    await waitFor(() => expect(buttonDisabled()).toBe(false));

    await act(async () => {
      screen.getByRole("button").click();
    });

    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/not registered with the Google client/i);
    });
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

  it("shows a backend refusal as an accessible alert", async () => {
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

    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/accounts can sign in with Google/i);
    });
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

  it("handles a One Tap response that carries no credential", async () => {
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
