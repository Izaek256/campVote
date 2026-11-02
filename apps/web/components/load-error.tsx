"use client";

import Link from "next/link";
import { Icon } from "./icon";

/**
 * Replaces a page's body when its initial load failed. The toast reports the
 * failure once, but a toast is gone in seconds, so this keeps the reason on
 * screen with a way to retry. Without it a failed load would leave the loading
 * skeleton showing for ever.
 */
export function LoadError({
  title,
  message,
  tone = "bad",
  backHref,
  backLabel = "Go back",
  action,
}: {
  title: string;
  message: string;
  tone?: "bad" | "hatch";
  backHref?: string;
  backLabel?: string;
  /** Replaces the default "Try again" button, e.g. with a sign-in link. */
  action?: React.ReactNode;
}) {
  return (
    <div className="panel stack">
      <div className={`banner ${tone}`} role="alert">
        <Icon name="alert" />
        <div>
          <b>{title}</b>
          {message}
        </div>
      </div>
      <div className="row">
        {action ?? (
          <button type="button" className="btn" onClick={() => window.location.reload()}>
            Try again
          </button>
        )}
        {backHref ? (
          <Link className="btn ghost" href={backHref}>
            {backLabel}
          </Link>
        ) : null}
      </div>
    </div>
  );
}
