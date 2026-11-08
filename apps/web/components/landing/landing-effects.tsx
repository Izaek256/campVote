"use client";

import { useEffect } from "react";

/**
 * The landing page's scroll behaviour, ported from the prototype:
 * reading progress, the docking/shrinking navigation, the sliding section
 * underline, the navigation's dark state over the security band, reveal-on-
 * scroll, and the back-to-top ring. DOM work instead of React state keeps
 * scrolling off the render path. Reduced motion is handled entirely in CSS.
 */
export function LandingEffects() {
  useEffect(() => {
    const progress = document.getElementById("lp-prog");
    const ring = document.getElementById("lp-ring");
    const nav = document.querySelector<HTMLElement>(".lp-nav");
    const up = document.querySelector<HTMLButtonElement>(".lp-up");
    const indicator = document.querySelector<HTMLElement>(".lp-ind");
    const links = [...document.querySelectorAll<HTMLAnchorElement>(".lp-lk a")];
    const sections = [...document.querySelectorAll<HTMLElement>(".lp-sec[id]")];
    const band = document.getElementById("security");
    const menu = document.querySelector<HTMLButtonElement>(".lp-mb");

    const onScroll = () => {
      const height = document.documentElement.scrollHeight - window.innerHeight;
      const ratio = height > 0 ? Math.min(1, window.scrollY / height) : 0;
      if (progress) progress.style.transform = `scaleX(${ratio})`;
      if (ring) ring.style.strokeDashoffset = String(100 - ratio * 100);
      nav?.classList.toggle("lp-s", window.scrollY > 40);
      up?.classList.toggle("lp-on", window.scrollY > 600);
    };

    const spy = () => {
      const middle = window.innerHeight * 0.4;
      let active: string | null = null;
      for (const section of sections) {
        const rect = section.getBoundingClientRect();
        if (rect.top <= middle && rect.bottom > middle) active = section.id;
      }
      const current = links.find((link) => link.hash === `#${active}`);
      for (const link of links) link.classList.toggle("lp-on", link === current);
      if (indicator) {
        if (current && window.innerWidth > 760) {
          indicator.style.opacity = "1";
          indicator.style.width = `${current.offsetWidth - 24}px`;
          indicator.style.transform = `translateX(${current.offsetLeft + 12}px)`;
        } else {
          indicator.style.opacity = "0";
        }
      }
      if (band && nav) {
        const rect = band.getBoundingClientRect();
        nav.classList.toggle("lp-dk", rect.top <= 40 && rect.bottom > 40);
      }
    };

    const onScrollBoth = () => {
      onScroll();
      spy();
    };

    onScrollBoth();
    window.addEventListener("scroll", onScrollBoth, { passive: true });
    window.addEventListener("resize", spy);

    const reveal = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          entry.target.classList.add("lp-in");
          reveal.unobserve(entry.target);
        }
      },
      { threshold: 0.15 }
    );
    const softReveal = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          entry.target.classList.add("lp-sx-in");
          softReveal.unobserve(entry.target);
        }
      },
      { rootMargin: "0px 0px -12% 0px" }
    );
    document.querySelectorAll(".lp-rv").forEach((el) => {
      reveal.observe(el);
    });
    document.querySelectorAll(".lp-sx").forEach((el) => {
      softReveal.observe(el);
    });

    const top = () => window.scrollTo({ top: 0, behavior: "smooth" });
    up?.addEventListener("click", top);

    const closeMenu = () => {
      nav?.classList.remove("lp-m");
      menu?.setAttribute("aria-expanded", "false");
    };
    links.forEach((link) => {
      link.addEventListener("click", closeMenu);
    });

    // The theme toggle swaps data-theme; let colour transitions ease instead
    // of snapping, then drop the helper class so hovers stay immediate.
    let themeTimer = 0;
    const observer = new MutationObserver(() => {
      const root = document.documentElement;
      root.classList.add("lp-tr");
      window.clearTimeout(themeTimer);
      themeTimer = window.setTimeout(() => root.classList.remove("lp-tr"), 600);
    });
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-theme"],
    });

    return () => {
      window.removeEventListener("scroll", onScrollBoth);
      window.removeEventListener("resize", spy);
      reveal.disconnect();
      softReveal.disconnect();
      observer.disconnect();
      up?.removeEventListener("click", top);
      links.forEach((link) => {
        link.removeEventListener("click", closeMenu);
      });
      window.clearTimeout(themeTimer);
    };
  }, []);

  return (
    <>
      <div id="lp-prog" aria-hidden="true" />
      <button type="button" className="lp-up" aria-label="Back to top">
        <svg viewBox="0 0 52 52" aria-hidden="true">
          <rect
            x="1.5"
            y="1.5"
            width="49"
            height="49"
            pathLength={100}
            stroke="var(--line)"
            strokeWidth={3}
          />
          <rect
            id="lp-ring"
            x="1.5"
            y="1.5"
            width="49"
            height="49"
            pathLength={100}
            stroke="var(--primary)"
            strokeWidth={3}
            strokeDasharray="100"
            strokeDashoffset="100"
          />
        </svg>
        <svg
          viewBox="0 0 24 24"
          width="20"
          height="20"
          aria-hidden="true"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path d="M12 20V5m0 0-6 6m6-6 6 6" />
        </svg>
      </button>
    </>
  );
}
