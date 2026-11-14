// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SiteHeader } from "@/components/site-header";

vi.mock("next/navigation", () => ({
  usePathname: () => "/",
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock("@/components/auth-provider", () => ({
  isAdmin: () => false,
  isStaff: () => false,
  useAuth: () => ({
    user: { first_name: "Student", last_name: "Voter" },
    signOut: vi.fn(),
  }),
}));

vi.mock("@/components/theme-toggle", () => ({
  ThemeToggle: () => null,
}));

afterEach(cleanup);

describe("SiteHeader", () => {
  it("provides an accessible toggle for the compact navigation", () => {
    render(<SiteHeader />);

    const toggle = screen.getByRole("button", { name: "Open navigation menu" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(toggle.getAttribute("aria-controls")).toBe("site-header-menu");

    fireEvent.click(toggle);

    expect(
      screen.getByRole("button", { name: "Close navigation menu" }).getAttribute("aria-expanded")
    ).toBe("true");
    const navigation = screen.getByRole("navigation", { name: "Main" });
    expect(within(navigation).getByRole("link", { name: "Your account" })).toBeTruthy();

    fireEvent.click(screen.getByRole("link", { name: "How voting works" }));
    expect(
      screen.getByRole("button", { name: "Open navigation menu" }).getAttribute("aria-expanded")
    ).toBe("false");
  });
});
