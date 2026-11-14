"use client";

import { useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, residences } from "@/lib/api";
import type { ResidenceArea } from "@/lib/types";

function Areas() {
  const toast = useToast();
  const [areas, setAreas] = useState<ResidenceArea[] | null>(null);
  const [loadError, setLoadError] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState<ResidenceArea | null>(null);

  const [email, setEmail] = useState("");
  const [correctArea, setCorrectArea] = useState("");
  const [correctBusy, setCorrectBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setAreas(await residences.list());
      setLoadError("");
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Residences could not be loaded.";
      setLoadError(message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function add(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      await residences.create(name.trim());
      setName("");
      await load();
      toast.success("Residence area added.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The residence could not be added.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!pending) return;
    setBusy(true);
    try {
      await residences.remove(pending.id);
      toast.success("Residence area deactivated and cleared from profiles.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The residence could not be removed.");
    } finally {
      setBusy(false);
      setPending(null);
      await load();
    }
  }

  async function correct(event: React.FormEvent) {
    event.preventDefault();
    setCorrectBusy(true);
    try {
      await residences.correct(email.trim(), correctArea || null);
      toast.success("Residence corrected.");
      setEmail("");
      setCorrectArea("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The residence could not be corrected.");
    } finally {
      setCorrectBusy(false);
    }
  }

  return (
    <StaffShell title="Residence areas">
      <div className="page-head" style={{ marginBottom: 20 }}>
        <div>
          <span className="lbl">Campus profile</span>
          <h1>Residence areas</h1>
          <p>
            Students choose their residence from this list before voting. The list starts empty —
            add the areas this campus actually uses. Removing an area deactivates it and clears it
            from the profiles that used it.
          </p>
        </div>
      </div>

      <form className="panel stack" onSubmit={add} style={{ marginBottom: 24 }}>
        <div className="section-h" style={{ marginTop: 0 }}>
          <h2>Add an area</h2>
        </div>
        <div className="field">
          <label htmlFor="area-name">Residence name</label>
          <input
            id="area-name"
            className="in"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="e.g. Hall 3"
            minLength={2}
            maxLength={120}
            required
          />
        </div>
        <div className="row">
          <button className="btn primary" type="submit" disabled={busy}>
            {busy ? "Saving…" : "Add area"}
          </button>
        </div>
      </form>

      {loadError ? <LoadError title="We could not load residences" message={loadError} /> : null}

      {!areas && !loadError ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 14 }} />
          <span className="skel" style={{ height: 14, width: "70%" }} />
        </div>
      ) : null}

      {areas ? (
        <div className="tbl-wrap panel pad0" style={{ marginBottom: 24 }}>
          <table className="tbl">
            <thead>
              <tr>
                <th>Residence</th>
                <th aria-label="Actions" />
              </tr>
            </thead>
            <tbody>
              {areas.length === 0 ? (
                <tr>
                  <td colSpan={2}>
                    No residence areas yet. Add the first one above so students can complete their
                    profile.
                  </td>
                </tr>
              ) : (
                areas.map((area) => (
                  <tr key={area.id}>
                    <td>
                      <b>{area.name}</b>
                    </td>
                    <td>
                      <div className="table-actions">
                        <button
                          type="button"
                          className="btn sm danger-o"
                          onClick={() => setPending(area)}
                        >
                          Remove
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      ) : null}

      <form className="panel stack" onSubmit={correct}>
        <div className="section-h" style={{ marginTop: 0 }}>
          <h2>Correct a student&apos;s residence</h2>
          <span className="mono">residence only</span>
        </div>
        <p className="hint" style={{ marginTop: 0 }}>
          Age and sex are recorded by the student and cannot be changed here. Leave the area empty
          to clear it.
        </p>
        <div className="field">
          <label htmlFor="correction-email">Student email</label>
          <input
            id="correction-email"
            className="in"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </div>
        <div className="field">
          <label htmlFor="correction-area">Residence area</label>
          <select
            id="correction-area"
            className="sel"
            value={correctArea}
            onChange={(event) => setCorrectArea(event.target.value)}
          >
            <option value="">Clear the residence</option>
            {(areas ?? []).map((area) => (
              <option key={area.id} value={area.id}>
                {area.name}
              </option>
            ))}
          </select>
        </div>
        <div className="row">
          <button className="btn" type="submit" disabled={correctBusy}>
            <Icon name="user" size="s" />
            {correctBusy ? "Saving…" : "Correct residence"}
          </button>
        </div>
      </form>

      <ConfirmDialog
        open={!!pending}
        title="Deactivate this residence?"
        confirmLabel="Deactivate"
        destructive
        busy={busy}
        onConfirm={remove}
        onClose={() => setPending(null)}
      >
        <p>
          <b>{pending?.name}</b> will disappear from the profile form, and every student assigned to
          it will have to choose again. Nothing is deleted.
        </p>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function AreasPage() {
  return (
    <RequireAuth adminOnly>
      <Areas />
    </RequireAuth>
  );
}
