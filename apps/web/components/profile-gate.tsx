"use client";

import { useEffect, useRef, useState } from "react";
import { ApiError, account, residences } from "@/lib/api";
import type { ResidenceArea } from "@/lib/types";
import { isAdmin, useAuth } from "./auth-provider";

/**
 * The voting profile every student fills once: age, sex and campus residence.
 *
 * The modal is blocking while it can be completed, because the server refuses a
 * ballot without these fields. When no residence areas exist yet there is
 * nothing to complete, so it offers "Later" instead of trapping the student on
 * a page they cannot leave.
 */
export function ProfileGate() {
  const { user, loading, refresh } = useAuth();
  const ref = useRef<HTMLDialogElement>(null);
  const [areas, setAreas] = useState<ResidenceArea[] | null>(null);
  const [age, setAge] = useState("");
  const [sex, setSex] = useState<"" | "MALE" | "FEMALE">("");
  const [areaId, setAreaId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [dismissed, setDismissed] = useState(false);

  const missingAge = !!user && user.age == null;
  const missingSex = !!user && !user.sex;
  const missingResidence = !!user && !user.residence_area_id;
  const profileMissing = !!user && !isAdmin(user) && (missingAge || missingSex || missingResidence);
  const incomplete = profileMissing && !dismissed;

  useEffect(() => {
    if (!incomplete) return;
    const dialog = ref.current;
    if (dialog && !dialog.open) dialog.showModal();
  }, [incomplete]);

  // Residence areas are maintained by an administrator while the student may
  // already be signed in. Re-read them while the profile is missing and when
  // the tab regains focus, and drop an earlier "Later" as soon as the form can
  // be completed: the profile must be captured, not skipped once.
  useEffect(() => {
    if (!profileMissing) return;
    function load() {
      residences
        .list()
        .then((list) => {
          setAreas(list);
          if (list.length > 0) setDismissed(false);
        })
        .catch(() => setAreas([]));
    }
    load();
    window.addEventListener("focus", load);
    return () => window.removeEventListener("focus", load);
  }, [profileMissing]);

  // A different account starts with a clean form.
  const seenUser = useRef<string | null>(null);
  useEffect(() => {
    const id = user?.id ?? null;
    if (id === seenUser.current) return;
    seenUser.current = id;
    setDismissed(false);
    setAreas(null);
  }, [user?.id]);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (missingAge) {
      const parsed = Number(age);
      if (!Number.isInteger(parsed) || parsed < 16 || parsed > 100) {
        setError("Age must be a whole number between 16 and 100.");
        return;
      }
    }
    if (missingSex && !sex) {
      setError("Choose MALE or FEMALE.");
      return;
    }
    if (missingResidence && !areaId) {
      setError("Choose your residence area.");
      return;
    }

    setBusy(true);
    try {
      await account.update({
        ...(missingAge ? { age: Number(age) } : {}),
        ...(missingSex && sex ? { sex } : {}),
        ...(missingResidence && areaId ? { residence_area_id: areaId } : {}),
      });
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError || err instanceof Error ? err.message : "Try again.");
    } finally {
      setBusy(false);
    }
  }

  if (loading || !incomplete) return null;
  const noAreas = areas !== null && areas.length === 0;

  return (
    <dialog
      ref={ref}
      className="modal-dialog modal"
      aria-labelledby="profile-gate-title"
      onCancel={(event) => event.preventDefault()}
    >
      <div className="modal-h">
        <h3 id="profile-gate-title">Complete your voting profile</h3>
      </div>
      <form onSubmit={save}>
        <div className="modal-b">
          <p className="hint" style={{ marginTop: 0 }}>
            These details are required before a ballot can be issued. They are kept with your
            account, never with your ballot.
          </p>

          {missingAge ? (
            <div className="field">
              <label htmlFor="profile-age">Age</label>
              <input
                id="profile-age"
                className="in"
                type="number"
                inputMode="numeric"
                min={16}
                max={100}
                step={1}
                value={age}
                onChange={(event) => setAge(event.target.value)}
                required
              />
              <span className="hint">Between 16 and 100. This cannot be changed once saved.</span>
            </div>
          ) : null}

          {missingSex ? (
            <div className="field">
              <label htmlFor="profile-sex">Sex</label>
              <select
                id="profile-sex"
                className="sel"
                value={sex}
                onChange={(event) => setSex(event.target.value as "" | "MALE" | "FEMALE")}
                required
              >
                <option value="">Select…</option>
                <option value="MALE">Male</option>
                <option value="FEMALE">Female</option>
              </select>
              <span className="hint">This cannot be changed once saved.</span>
            </div>
          ) : null}

          {missingResidence ? (
            <div className="field">
              <label htmlFor="profile-residence">Residence</label>
              {noAreas ? (
                <span className="err" role="status">
                  No residence areas have been set up yet. An election administrator must add them
                  before you can vote.
                </span>
              ) : (
                <select
                  id="profile-residence"
                  className="sel"
                  value={areaId}
                  onChange={(event) => setAreaId(event.target.value)}
                  required
                >
                  <option value="">Select…</option>
                  {(areas ?? []).map((area) => (
                    <option key={area.id} value={area.id}>
                      {area.name}
                    </option>
                  ))}
                </select>
              )}
              <span className="hint">
                A wrong residence is corrected by an election administrator, not here.
              </span>
            </div>
          ) : null}

          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
        </div>
        <div className="modal-f">
          {noAreas ? (
            <button type="button" className="btn ghost" onClick={() => setDismissed(true)}>
              Later
            </button>
          ) : null}
          <button className="btn primary" type="submit" disabled={busy || noAreas}>
            {busy ? "Saving…" : "Save profile"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
