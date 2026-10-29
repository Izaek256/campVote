"use client";

import { useEffect, useRef } from "react";

/**
 * Native <dialog> gives focus containment, Escape to close and focus return
 * without a hand-written trap.
 */
export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  busy,
  destructive,
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  children: React.ReactNode;
  confirmLabel: string;
  busy?: boolean;
  destructive?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className="modal-dialog modal"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      onClose={onClose}
    >
      <div className="modal-h">
        <h3>{title}</h3>
      </div>
      <div className="modal-b">{children}</div>
      <div className="modal-f">
        <button type="button" className="btn ghost" onClick={onClose} disabled={busy}>
          Cancel
        </button>
        <button
          type="button"
          className={`btn ${destructive ? "danger" : "primary"}`}
          onClick={onConfirm}
          disabled={busy}
        >
          {busy ? "Working…" : confirmLabel}
        </button>
      </div>
    </dialog>
  );
}
