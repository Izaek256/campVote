"use client";

import { useEffect, useState } from "react";
import { Icon } from "./icon";

type Theme = "light" | "dark";

/**
 * Returns the theme the user is actually seeing right now. When no explicit
 * theme is set on the root element, the CSS media query decides, so we ask
 * the browser.
 */
function currentTheme(): Theme {
  if (typeof window === "undefined") return "light";
  const attr = document.documentElement.getAttribute("data-theme");
  if (attr === "dark" || attr === "light") return attr;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("light");
  const [mounted, setMounted] = useState(false);

  // Sync the React state with the stored or system-default theme on mount.
  useEffect(() => {
    setMounted(true);
    const saved = localStorage.getItem("theme");
    if (saved === "light" || saved === "dark") {
      setTheme(saved);
    } else {
      setTheme(currentTheme());
    }
  }, []);

  // Apply the theme to the DOM whenever it changes.
  useEffect(() => {
    if (!mounted) return;
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("theme", theme);
  }, [theme, mounted]);

  if (!mounted) {
    return (
      <button type="button" className="btn ghost sm" aria-label="Toggle theme" disabled>
        <Icon name="moon" size="s" />
      </button>
    );
  }

  return (
    <button
      type="button"
      className="btn ghost sm"
      onClick={() => setTheme((prev) => (prev === "dark" ? "light" : "dark"))}
      aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
      title={`Theme: ${theme}`}
    >
      <Icon name={theme === "dark" ? "sun" : "moon"} size="s" />
    </button>
  );
}
