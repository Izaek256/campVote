"use client";

import { useEffect, useRef } from "react";
import { X } from "@/lib/icons";

type Variant = "danger" | "primary";

interface ConfirmModalProps {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: Variant;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * A simple, accessible confirmation dialog.
 *
 * Traps focus when open and closes on Escape. Uses the native dialog element
 * so that browsers and screen readers handle the overlay role automatically.
 */
export function ConfirmModal({
  open,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "danger",
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);

  // Sync <dialog> open state imperatively — the HTML attribute approach is
  // cleaner than a portal and keeps the component self-contained.
  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    if (open) {
      if (!el.open) el.showModal();
      // Give the confirm button focus so keyboard users can act immediately.
      setTimeout(() => confirmRef.current?.focus(), 50);
    } else {
      if (el.open) el.close();
    }
  }, [open]);

  // Let the browser's <dialog> handle Escape key natively; we also listen so
  // we can synchronise React state.
  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    const handleCancel = (e: Event) => {
      e.preventDefault();
      onCancel();
    };
    el.addEventListener("cancel", handleCancel);
    return () => el.removeEventListener("cancel", handleCancel);
  }, [onCancel]);

  const confirmButtonClass =
    variant === "danger"
      ? "rounded-lg bg-danger px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-danger focus:ring-offset-2"
      : "rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2";

  return (
    <dialog
      ref={dialogRef}
      className="m-auto w-full max-w-md rounded-xl border border-border bg-bg-elevated p-0 shadow-xl backdrop:bg-black/60"
      aria-labelledby="confirm-modal-title"
      aria-describedby={description ? "confirm-modal-desc" : undefined}
    >
      {/* Header */}
      <div className="flex items-start justify-between border-b border-border px-6 py-4">
        <h2 id="confirm-modal-title" className="text-base font-semibold text-foreground">
          {title}
        </h2>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg p-1 text-text-muted transition-colors hover:bg-bg-hover hover:text-foreground"
          aria-label="Close dialog"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>

      {/* Body */}
      {description ? (
        <div className="px-6 py-4">
          <p id="confirm-modal-desc" className="text-sm text-text-secondary">
            {description}
          </p>
        </div>
      ) : null}

      {/* Footer */}
      <div className="flex justify-end gap-2 border-t border-border bg-bg-secondary px-6 py-4">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-border px-4 py-2 text-sm text-text-secondary transition-colors hover:bg-bg-hover hover:text-foreground"
        >
          {cancelLabel}
        </button>
        <button ref={confirmRef} type="button" onClick={onConfirm} className={confirmButtonClass}>
          {confirmLabel}
        </button>
      </div>
    </dialog>
  );
}
