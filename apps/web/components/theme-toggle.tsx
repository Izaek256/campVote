"use client";

import { useEffect, useState } from "react";
import { Icon } from "./icon";

type Theme = "light" | "dark" | "system";

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("system");
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const saved = localStorage.getItem("theme") as Theme | null;
    if (saved) {
      setTheme(saved);
    }
  }, []);

  useEffect(() => {
    if (!mounted) return;

    const root = document.documentElement;
    root.removeAttribute("data-theme");

    if (theme === "system") {
      // Let system preference handle it via CSS media query
    } else {
      root.setAttribute("data-theme", theme);
    }

    localStorage.setItem("theme", theme);
  }, [theme, mounted]);

  if (!mounted) {
    return (
      <button
        type="button"
        className="btn ghost sm"
        aria-label="Toggle theme"
        disabled
      >
        <Icon name="moon" size="s" />
      </button>
    );
  }

  return (
    <button
      type="button"
      className="btn ghost sm"
      onClick={() => {
        setTheme((prev) => {
          if (prev === "system") return "light";
          if (prev === "light") return "dark";
          return "system";
        });
      }}
      aria-label={`Current theme: ${theme}. Click to cycle through themes`}
      title={`Theme: ${theme}`}
    >
      {theme === "dark" ? <Icon name="sun" size="s" /> : <Icon name="moon" size="s" />}
    </button>
  );
}
