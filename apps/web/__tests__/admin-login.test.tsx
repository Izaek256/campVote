// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import AdminLoginPage from "@/app/admin/login/page";

const mocks = vi.hoisted(() => ({
  replace: vi.fn(),
  signInAdmin: vi.fn(),
  verifyAdminSecondFactor: vi.fn(),
  user: null as { role: string } | null,
  loading: false,
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mocks.replace }),
}));

vi.mock("@/components/auth-provider", () => ({
  useAuth: () => ({
    user: mocks.user,
    loading: mocks.loading,
    signInAdmin: mocks.signInAdmin,
    verifyAdminSecondFactor: mocks.verifyAdminSecondFactor,
  }),
}));

vi.mock("@/components/theme-toggle", () => ({
  ThemeToggle: () => null,
}));

function renderPage() {
  return render(<AdminLoginPage />);
}

afterEach(cleanup);

beforeEach(() => {
  mocks.replace.mockReset();
  mocks.signInAdmin.mockReset();
  mocks.verifyAdminSecondFactor.mockReset();
  mocks.signInAdmin.mockResolvedValue({ requiresSecondFactor: false });
  mocks.verifyAdminSecondFactor.mockResolvedValue(undefined);
  mocks.user = null;
  mocks.loading = false;
});

describe("AdminLoginPage", () => {
  it("requires the configured staff email domain before sending credentials", async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText("University email"), {
      target: { value: "student@std.must.ac.ug" },
    });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect((await screen.findByRole("alert")).textContent).toMatch(/ending in @must\.ac\.ug/);
    expect(mocks.signInAdmin).not.toHaveBeenCalled();
  });

  it("normalizes a valid staff email and redirects after password sign-in", async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText("University email"), {
      target: { value: "Admin@MUST.ac.ug " },
    });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(mocks.signInAdmin).toHaveBeenCalledWith("admin@must.ac.ug", "password");
      expect(mocks.replace).toHaveBeenCalledWith("/staff");
    });
  });

  it("completes the authenticator challenge before redirecting", async () => {
    mocks.signInAdmin.mockResolvedValue({
      requiresSecondFactor: true,
      pendingToken: "pending-token",
      method: "authenticator",
    });
    renderPage();
    fireEvent.change(screen.getByLabelText("University email"), {
      target: { value: "admin@must.ac.ug" },
    });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    const codeInput = await screen.findByLabelText("Verification code");
    fireEvent.change(codeInput, { target: { value: "123456" } });
    fireEvent.click(screen.getByRole("button", { name: "Verify and sign in" }));

    await waitFor(() => {
      expect(mocks.verifyAdminSecondFactor).toHaveBeenCalledWith("pending-token", "123456", false);
      expect(mocks.replace).toHaveBeenCalledWith("/staff");
    });
  });

  it("supports backup codes as an alternative second factor", async () => {
    mocks.signInAdmin.mockResolvedValue({
      requiresSecondFactor: true,
      pendingToken: "pending-token",
      method: "authenticator",
    });
    renderPage();
    fireEvent.change(screen.getByLabelText("University email"), {
      target: { value: "admin@must.ac.ug" },
    });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await screen.findByRole("heading", { name: "Verify it's you" });

    fireEvent.click(screen.getByRole("button", { name: "Use a backup code" }));
    fireEvent.change(screen.getByLabelText("Backup code"), { target: { value: "backup-code" } });
    fireEvent.click(screen.getByRole("button", { name: "Verify and sign in" }));

    await waitFor(() => {
      expect(mocks.verifyAdminSecondFactor).toHaveBeenCalledWith("pending-token", "backup-code", true);
      expect(mocks.replace).toHaveBeenCalledWith("/staff");
    });
  });

  it("shows request references when sign-in fails", async () => {
    const { ApiError } = await import("@/lib/api");
    mocks.signInAdmin.mockRejectedValue(
      new ApiError("The service is unavailable", 503, {
        code: "SERVICE_UNAVAILABLE",
        requestId: "req-test-123",
      })
    );
    renderPage();
    fireEvent.change(screen.getByLabelText("University email"), {
      target: { value: "admin@must.ac.ug" },
    });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect((await screen.findByText("Request ID: req-test-123")).textContent).toBe(
      "Request ID: req-test-123"
    );
  });
});
