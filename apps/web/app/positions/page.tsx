"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { useAuth } from "@/components/auth-provider";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { useToast } from "@/components/toast";
import { FilePicker } from "@/components/file-picker";
import { ApiError, positions, fileUrl, upload } from "@/lib/api";
import type { EligiblePosition } from "@/lib/types";

function eligibilitySummary(position: EligiblePosition): string {
  const parts: string[] = [];
  if (position.allowed_domain) parts.push(`@${position.allowed_domain}`);
  if (position.allowed_years) parts.push(`years ${position.allowed_years}`);
  if (position.allowed_courses) parts.push(`courses ${position.allowed_courses}`);
  return parts.length ? `You qualify as ${parts.join(" · ")}` : "Open to any student";
}

/** Age in whole years from a YYYY-MM-DD date, or null while it is unreadable. */
function ageFrom(dateOfBirth: string): number | null {
  if (!dateOfBirth) return null;
  const born = new Date(`${dateOfBirth}T00:00:00Z`);
  if (Number.isNaN(born.getTime())) return null;
  const now = new Date();
  let age = now.getUTCFullYear() - born.getUTCFullYear();
  const monthDiff = now.getUTCMonth() - born.getUTCMonth();
  if (monthDiff < 0 || (monthDiff === 0 && now.getUTCDate() < born.getUTCDate())) age -= 1;
  return age >= 0 && age < 130 ? age : null;
}

type ApplicationForm = {
  full_name: string;
  date_of_birth: string;
  sex: string;
  religion: string;
  campus_residence: string;
  slogan: string;
  bio: string;
  passport_photo_url: string;
  partial_transcript_url: string;
};

const emptyApplication: ApplicationForm = {
  full_name: "",
  date_of_birth: "",
  sex: "",
  religion: "",
  campus_residence: "",
  slogan: "",
  bio: "",
  passport_photo_url: "",
  partial_transcript_url: "",
};

function RunForOffice() {
  const { user } = useAuth();
  const [items, setItems] = useState<EligiblePosition[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const toast = useToast();
  const [applying, setApplying] = useState<EligiblePosition | null>(null);
  const [form, setForm] = useState<ApplicationForm>(emptyApplication);

  const load = useCallback(async () => {
    try {
      const data = await positions.eligible();
      setItems(data.positions ?? []);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Positions could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  function startApply(position: EligiblePosition) {
    setApplying(position);
    setForm({
      ...emptyApplication,
      full_name: user ? `${user.first_name} ${user.last_name}` : "",
    });
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!applying) return;
    if (!form.date_of_birth) {
      toast.error("Enter your date of birth.");
      return;
    }
    if (!form.sex) {
      toast.error("Choose your sex.");
      return;
    }
    if (!form.passport_photo_url) {
      toast.error("Upload your passport photo.");
      return;
    }
    if (!form.partial_transcript_url) {
      toast.error("Upload your partial transcript.");
      return;
    }
    setBusy(true);
    try {
      await positions.apply(applying.id, form);
      toast.success(`Your application for ${applying.title} has been submitted for review.`);
      setApplying(null);
      await load();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Your application could not be submitted."
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <AppShell title="Run for a position">
      <div className="page-head">
        <div>
          <span className="lbl">Candidate</span>
          <h1>Run for a position</h1>
          <p>These are the positions you are eligible to stand for. Apply and wait for review.</p>
        </div>
        <Link className="btn" href="/candidate">
          <Icon name="user" size="s" />
          My candidacy
        </Link>
      </div>

      {applying ? (
        <form className="panel stack" onSubmit={submit} style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Apply for {applying.title}</h2>
            <p>{eligibilitySummary(applying)}</p>
          </div>
          <div className="form-grid">
            <div className="field full">
              <label htmlFor="apply-name">Names to appear on the ballot *</label>
              <input
                id="apply-name"
                className="in"
                value={form.full_name}
                onChange={(event) => setForm({ ...form, full_name: event.target.value })}
                required
              />
            </div>
            <div className="field">
              <label htmlFor="apply-dob">Date of birth *</label>
              <input
                id="apply-dob"
                className="in"
                type="date"
                value={form.date_of_birth}
                onChange={(event) => setForm({ ...form, date_of_birth: event.target.value })}
                required
              />
            </div>
            <div className="field">
              <label htmlFor="apply-age">Age</label>
              <input
                id="apply-age"
                className="in"
                value={ageFrom(form.date_of_birth) ?? ""}
                readOnly
                placeholder="—"
              />
              <span className="hint">Calculated from your date of birth.</span>
            </div>
            <div className="field">
              <label htmlFor="apply-sex">Sex *</label>
              <select
                id="apply-sex"
                className="sel"
                value={form.sex}
                onChange={(event) => setForm({ ...form, sex: event.target.value })}
                required
              >
                <option value="">Choose…</option>
                <option value="MALE">Male</option>
                <option value="FEMALE">Female</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="apply-religion">Religion</label>
              <input
                id="apply-religion"
                className="in"
                value={form.religion}
                onChange={(event) => setForm({ ...form, religion: event.target.value })}
              />
            </div>
            <div className="field full">
              <label htmlFor="apply-residence">Campus residence / address</label>
              <input
                id="apply-residence"
                className="in"
                value={form.campus_residence}
                onChange={(event) => setForm({ ...form, campus_residence: event.target.value })}
                placeholder="Hall of residence or where you stay"
              />
            </div>
            <div className="field full">
              <label htmlFor="apply-slogan">Slogan</label>
              <input
                id="apply-slogan"
                className="in"
                value={form.slogan}
                maxLength={200}
                onChange={(event) => setForm({ ...form, slogan: event.target.value })}
                placeholder="A short line voters will see"
              />
            </div>
            <div className="field full">
              <label htmlFor="apply-bio">Why you are standing</label>
              <textarea
                id="apply-bio"
                className="ta"
                rows={5}
                value={form.bio}
                onChange={(event) => setForm({ ...form, bio: event.target.value })}
              />
            </div>
            <FilePicker
              accepts="image"
              maxSize={2 * 1024 * 1024}
              value={form.passport_photo_url}
              onChange={(url) => setForm({ ...form, passport_photo_url: url })}
              label="Passport photo *"
              hint="JPEG, PNG, GIF, WebP · max 2 MB"
            />
            <FilePicker
              accepts="pdf"
              maxSize={5 * 1024 * 1024}
              value={form.partial_transcript_url}
              onChange={(url) => setForm({ ...form, partial_transcript_url: url })}
              label="Partial transcript *"
              hint="PDF · max 5 MB"
            />
          </div>
          <div className="row">
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? "Submitting…" : "Submit application"}
            </button>
            <button type="button" className="btn ghost" onClick={() => setApplying(null)}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {loadError ? (
        <LoadError
          title="We could not load the positions"
          message={loadError}
          backHref="/elections"
          backLabel="Back to elections"
        />
      ) : !items ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {items && items.length === 0 ? (
        <EmptyState title="No positions are open to you right now">
          When an administrator opens a position you qualify for, it will appear here.
        </EmptyState>
      ) : null}

      {items && items.length > 0 ? (
        <div className="grid">
          {items.map((position) => (
            <div key={position.id} className="panel stack">
              <div>
                <span className="lbl">{position.election_title || "Applications open"}</span>
                <h2 style={{ margin: "4px 0" }}>{position.title}</h2>
                {position.description ? <p>{position.description}</p> : null}
              </div>
              {position.requirements ? (
                <div>
                  <span className="lbl">Requirements</span>
                  <p style={{ whiteSpace: "pre-wrap" }}>{position.requirements}</p>
                </div>
              ) : null}
              <span className="candidate-meta">{eligibilitySummary(position)}</span>
              <div className="row">
                {position.applied ? (
                  <span className="chip draft">Applied — awaiting review</span>
                ) : (
                  <button
                    type="button"
                    className="btn primary"
                    onClick={() => startApply(position)}
                  >
                    Apply
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </AppShell>
  );
}

export default function PositionsPage() {
  return (
    <RequireAuth>
      <RunForOffice />
    </RequireAuth>
  );
}
