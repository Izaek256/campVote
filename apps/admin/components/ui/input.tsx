import type { InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  /** An explicit error message — renders a red ring when provided. */
  error?: string;
}

interface InputClassesOptions {
  className?: string;
  error?: boolean;
}

const base =
  "w-full rounded-lg border bg-bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-text-muted focus:outline-none focus:ring-1 transition-colors";
const normal = "border-border focus:border-accent focus:ring-accent";
const errored = "border-danger/50 focus:border-danger focus:ring-danger";

/**
 * Returns a class string for <input> elements. Accepts an options object
 * with an optional `className` to merge in and an optional `error` boolean.
 */
export function inputClasses({ className = "", error = false }: InputClassesOptions = {}): string {
  return [base, error ? errored : normal, className].join(" ").trim();
}

export function Input({ error, className = "", ...props }: InputProps) {
  return (
    <div className="flex flex-col gap-1">
      <input
        className={inputClasses({ error: Boolean(error), className })}
        aria-invalid={error ? "true" : undefined}
        {...props}
      />
      {error ? (
        <p role="alert" className="text-xs text-danger">
          {error}
        </p>
      ) : null}
    </div>
  );
}
