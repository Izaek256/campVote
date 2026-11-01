"use client";

import { useState } from "react";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { apiClient } from "@/lib/api-client";
import { getApiErrorMessage } from "@/lib/api-core";

interface DeleteAccountDialogProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Confirmation dialog for self-deleting an admin account.
 *
 * Warns clearly that this action is soft-deletable from the Deleted Accounts
 * page, not permanent.
 */
export function DeleteAccountDialog({ open, onClose }: DeleteAccountDialogProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    setBusy(true);
    setError(null);
    try {
      await apiClient.delete("/api/v1/profile");
      // Redirect to login — the session is now invalid.
      window.location.href = "/login";
    } catch (err) {
      setError(getApiErrorMessage(err, "Could not delete your account. Please try again."));
      setBusy(false);
    }
  }

  return (
    <>
      {error && (
        <p role="alert" className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
      <ConfirmModal
        open={open && !busy}
        title="Delete your account?"
        description="Your account will be closed and you will be signed out. An admin can restore it from the Deleted Accounts page within 30 days."
        confirmLabel="Delete my account"
        variant="danger"
        onConfirm={handleConfirm}
        onCancel={onClose}
      />
    </>
  );
}
