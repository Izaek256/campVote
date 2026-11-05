import { describe, expect, it } from "vitest";
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

  it("leaves a bucket URL (no /files path) untouched", () => {
    const bucket = "http://localhost:9002/uploads/photo.jpg";
    expect(fileUrl(bucket)).toBe(bucket);
  });
});
