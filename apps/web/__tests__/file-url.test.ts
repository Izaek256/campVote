import { describe, expect, it, vi } from "vitest";
import { API, fileUrl } from "@/lib/api";

describe("fileUrl", () => {
  it("returns an empty string for nothing uploaded", () => {
    expect(fileUrl("")).toBe("");
    expect(fileUrl(undefined)).toBe("");
    expect(fileUrl(null)).toBe("");
  });

  it("re-points a stored /files link at the API origin", () => {
    // The API wrote this link using APP_URL (the web app's port), where no
    // /files route exists. The browser must load it from the API instead.
    expect(fileUrl("http://localhost:3000/files/uploads/2026/11/photo.jpg")).toBe(
      `${API}/files/uploads/2026/11/photo.jpg`
    );
  });

  it("keeps a signed query string so the link stays valid", () => {
    expect(fileUrl("http://localhost:3000/files/uploads/a.pdf?expires=1&signature=ab")).toBe(
      `${API}/files/uploads/a.pdf?expires=1&signature=ab`
    );
  });

  it("resolves a site-relative path against the API", () => {
    expect(fileUrl("/files/uploads/photo.jpg")).toBe(`${API}/files/uploads/photo.jpg`);
  });

  it("keeps stored links on this origin when no API URL is configured", async () => {
    // Deployments behind the /api proxy leave NEXT_PUBLIC_API_URL unset, so
    // file links must resolve against the current origin, not localhost.
    vi.stubEnv("NEXT_PUBLIC_API_URL", "");
    vi.resetModules();
    const { fileUrl: sameOrigin } = await import("@/lib/api");
    expect(sameOrigin("http://194.36.88.77:8080/files/uploads/a.jpg")).toBe("/files/uploads/a.jpg");
    expect(sameOrigin("/files/uploads/photo.jpg")).toBe("/files/uploads/photo.jpg");
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it("leaves a bucket URL (no /files path) untouched", () => {
    const bucket = "http://localhost:9002/uploads/photo.jpg";
    expect(fileUrl(bucket)).toBe(bucket);
  });
});
