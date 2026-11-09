import { afterEach, describe, expect, it, vi } from "vitest";
import { scrollIntoView } from "@/lib/scroll-into-view";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("scrollIntoView", () => {
  it("scrolls smoothly unless reduced motion is requested", () => {
    const element = { scrollIntoView: vi.fn() } as unknown as HTMLElement;
    vi.stubGlobal("window", {
      matchMedia: vi.fn().mockReturnValue({ matches: false }),
    });

    scrollIntoView(element);

    expect(element.scrollIntoView).toHaveBeenCalledWith({
      behavior: "smooth",
      block: "start",
    });
  });

  it("uses immediate scrolling for reduced motion", () => {
    const element = { scrollIntoView: vi.fn() } as unknown as HTMLElement;
    vi.stubGlobal("window", {
      matchMedia: vi.fn().mockReturnValue({ matches: true }),
    });

    scrollIntoView(element);

    expect(element.scrollIntoView).toHaveBeenCalledWith({
      behavior: "auto",
      block: "start",
    });
  });

  it("does nothing when there is no target element", () => {
    expect(() => scrollIntoView(null)).not.toThrow();
  });
});
