"use client";

import { RULES, passwordFailures, passwordStrength, STRENGTH_LEVELS } from "@/lib/password-rules";

interface PasswordStrengthProps {
  value: string;
  about?: string[];
  id?: string;
  className?: string;
}

/**
 * Strength bar + rule checklist for password fields.
 *
 * Uses only inline styles and the design-token CSS variables from globals.css
 * — the web app has no Tailwind, so no utility classes here.
 *
 * Renders nothing until the user has started typing.
 */
export function PasswordStrength({
  value,
  about = [],
  id = "password-strength",
  className = "",
}: PasswordStrengthProps) {
  const failed = passwordFailures(value, about);
  const strength = passwordStrength(value, about);

  if (!value) return null;

  return (
    <div
      id={id}
      className={className}
      style={{ display: "flex", flexDirection: "column", gap: 10, marginTop: 4 }}
    >
      {/* ── Strength bar ── */}
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        {/* Five filled segments */}
        <div aria-hidden="true" style={{ display: "flex", flex: 1, gap: 4 }}>
          {STRENGTH_LEVELS.map((level, i) => (
            <span
              key={level.label}
              style={{
                flex: 1,
                height: 4,
                borderRadius: 2,
                background: i <= strength.score ? strength.color : "var(--line)",
                transition: "background 0.2s",
              }}
            />
          ))}
        </div>
        {/* Label */}
        <span
          role="status"
          aria-live="polite"
          style={{
            fontSize: 12,
            fontWeight: 600,
            color: strength.color,
            fontFamily: "var(--font-mono)",
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
        >
          {strength.label}
        </span>
      </div>

      {/* ── Rule checklist ── */}
      <ul
        style={{
          listStyle: "none",
          margin: 0,
          padding: 0,
          display: "flex",
          flexDirection: "column",
          gap: 5,
        }}
      >
        {RULES.map((rule) => {
          const ok = !failed.includes(rule.id);
          return (
            <li
              key={rule.id}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                fontSize: 13,
                color: ok ? "var(--ink-2)" : "var(--ink-3)",
              }}
            >
              {/* Small tick/dot indicator */}
              <span
                aria-hidden="true"
                style={{
                  flexShrink: 0,
                  width: 16,
                  height: 16,
                  borderRadius: "50%",
                  border: ok ? "none" : "1.5px solid var(--line-strong)",
                  background: ok ? "var(--primary)" : "transparent",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                {ok && (
                  <svg
                    width="9"
                    height="9"
                    viewBox="0 0 12 10"
                    fill="none"
                    stroke="var(--on-primary)"
                    strokeWidth="2.2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M1 5l3 3 7-7" />
                  </svg>
                )}
              </span>
              <span>
                {rule.label}
                <span
                  style={{
                    position: "absolute",
                    width: 1,
                    height: 1,
                    overflow: "hidden",
                    clip: "rect(0 0 0 0)",
                  }}
                >
                  {ok ? ": met" : ": not met yet"}
                </span>
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
