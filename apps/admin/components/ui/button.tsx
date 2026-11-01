import type { ButtonHTMLAttributes } from "react";

export type Variant = "primary" | "secondary" | "danger" | "ghost" | "outline";
type Size = "sm" | "md" | "lg";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

interface ButtonClassesOptions {
  variant?: Variant;
  size?: Size;
  className?: string;
}

const variantMap: Record<Variant, string> = {
  primary: "bg-accent text-white border-transparent hover:opacity-90",
  secondary:
    "border-border text-text-secondary bg-bg-secondary hover:bg-bg-hover hover:text-foreground",
  danger: "bg-danger text-white border-transparent hover:bg-danger/90",
  ghost: "border-transparent text-text-secondary hover:bg-bg-hover hover:text-foreground",
  outline: "border-border text-foreground bg-transparent hover:bg-bg-hover",
};

const sizeMap: Record<Size, string> = {
  sm: "px-2.5 py-1 text-xs rounded-lg",
  md: "px-4 py-2 text-sm rounded-lg",
  lg: "px-5 py-2.5 text-base rounded-xl",
};

const base =
  "inline-flex items-center justify-center gap-2 font-medium border transition-colors disabled:opacity-50 disabled:pointer-events-none";

/**
 * Returns a stable Tailwind class string for button styling.
 *
 * Accepts either positional args (variant, size, extra) or a single options
 * object { variant, size, className } — both call forms exist in the codebase.
 */
export function buttonClasses(
  variantOrOptions: Variant | ButtonClassesOptions = "secondary",
  size: Size = "md",
  extra = ""
): string {
  if (typeof variantOrOptions === "object") {
    const opts = variantOrOptions;
    return [
      base,
      variantMap[opts.variant ?? "secondary"],
      sizeMap[opts.size ?? "md"],
      opts.className ?? "",
    ]
      .join(" ")
      .trim();
  }
  return [base, variantMap[variantOrOptions], sizeMap[size], extra].join(" ").trim();
}

export function Button({
  variant = "secondary",
  size = "md",
  className = "",
  children,
  ...props
}: ButtonProps) {
  return (
    <button type="button" className={buttonClasses(variant, size, className)} {...props}>
      {children}
    </button>
  );
}
