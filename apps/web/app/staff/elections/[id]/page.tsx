"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { use, useCallback, useEffect, useRef, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { isAdmin, useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { StatusChip } from "@/components/status-chip";
import { useToast } from "@/components/toast";
import { ApiError, candidates, elections, positions } from "@/lib/api";
import {
  categoryLabel,
  EAT,
  formatEAT,
  formatNumber,
  formatPercent,
  statusLabel,
} from "@/lib/format";
import type {
  Candidate,
  Election,
  ElectionRole,
  EligibilityReport,
  Position,
  Results,
  RosterEntry,
} from "@/lib/types";

function toEatInput(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const parts = new Intl.DateTimeFormat("sv-SE", {
    timeZone: EAT,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
  return parts.replace(" ", "T");
}

function toISO(value: string): string {
  return value ? `${value}:00+03:00` : "";
}

const pad = (value: number) => String(value).padStart(2, "0");

/** Add a length to a datetime-local value (both stay in the same local wall clock). */
function addMinutes(value: string, minutes: number): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  date.setMinutes(date.getMinutes() + minutes);
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}`;
}

const durationPresets = [
  { label: "2 minutes", minutes: 2 },
  { label: "1 hour", minutes: 60 },
  { label: "6 hours", minutes: 6 * 60 },
  { label: "20 hours", minutes: 20 * 60 },
  { label: "1 day", minutes: 24 * 60 },
  { label: "1 week", minutes: 7 * 24 * 60 },
];

function transitionLabel(to: string): string {
  switch (to) {
    case "CLOSED":
      return "Close early";
    default:
      return statusLabel(to as Election["status"]);
  }
}

function Manage({ id }: { id: string }) {
  const toast = useToast();
  const router = useRouter();
  const { user } = useAuth();
  const [loadError, setLoadError] = useState<string | null>(null);
  const [election, setElection] = useState<Election | null>(null);
  const [report, setReport] = useState<EligibilityReport | null>(null);
  const [results, setResults] = useState<Results | null>(null);
  const [notes, setNotes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [pendingTo, setPendingTo] = useState("");
  const [reason, setReason] = useState("");
  const [confirmCertify, setConfirmCertify] = useState(false);
  const [certifyReason, setCertifyReason] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [form, setForm] = useState({
    title: "",
    description: "",
    starts_at: "",
    ends_at: "",
    public_listing: false,
    reason: "",
  });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [eligibility, setEligibility] = useState({
    allowed_domain: "",
    allowed_years: "",
    allowed_courses: "",
    require_roster: false,
  });
  const [rosterList, setRosterList] = useState<RosterEntry[] | null>(null);
  const [rosterBusy, setRosterBusy] = useState(false);
  const rosterFileRef = useRef<HTMLInputElement>(null);
  const [readiness, setReadiness] = useState<{
    ready: boolean;
    problems: Record<string, string>;
  } | null>(null);
  const [roles, setRoles] = useState<ElectionRole[]>([]);
  const [positionList, setPositionList] = useState<Position[]>([]);
  const [candidateList, setCandidateList] = useState<Candidate[]>([]);
  const [roleEmail, setRoleEmail] = useState("");
  const [roleName, setRoleName] = useState("CHIEF_ELECTION_ADMIN");
  const [roleBusy, setRoleBusy] = useState(false);
  const [cloneForm, setCloneForm] = useState({ title: "", starts_at: "", ends_at: "" });
  const [cloneBusy, setCloneBusy] = useState(false);
  // Only the creator or a system administrator may hand out election roles.
  const canManageRoles = isAdmin(user) || (!!election && election.created_by_id === user?.id);

  const load = useCallback(async () => {
    const messages: string[] = [];
    try {
      const current = await elections.get(id);
      setElection(current);
      setForm({
        title: current.title,
        description: current.description ?? "",
        starts_at: toEatInput(current.starts_at),
        ends_at: toEatInput(current.ends_at),
        public_listing: current.public_listing,
        reason: "",
      });
      setEligibility({
        allowed_domain: current.allowed_domain ?? "",
        allowed_years: current.allowed_years ?? "",
        allowed_courses: current.allowed_courses ?? "",
        require_roster: current.require_roster ?? false,
      });
      setRosterList(
        current.require_roster ? await elections.roster.list(id).catch(() => null) : null
      );
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Election could not be loaded.";
      setLoadError(message);
      toast.error(message);
      return;
    }
    const reportResult = await elections.report(id).catch((err) => {
      messages.push(
        err instanceof ApiError && err.status === 403
          ? "Eligibility report needs the roster permission."
          : "Eligibility report could not be loaded."
      );
      return null;
    });
    setReport(reportResult);
    const resultsResult = await elections.results(id).catch((err) => {
      messages.push(
        err instanceof ApiError && err.status === 403
          ? "Results are not published for this election yet."
          : "Results could not be loaded."
      );
      return null;
    });
    setResults(resultsResult);
    const readinessResult = await elections.readiness(id).catch((err) => {
      messages.push(
        err instanceof ApiError && err.status === 403
          ? "Readiness checks need the election edit permission."
          : "Readiness could not be checked."
      );
      return null;
    });
    setReadiness(readinessResult);
    const roleResult = await elections.roles.list(id).catch((err) => {
      messages.push(
        err instanceof ApiError && err.status === 403
          ? "Role assignments need the elections.roles permission."
          : "Role assignments could not be loaded."
      );
      return [] as ElectionRole[];
    });
    setRoles(roleResult ?? []);
    // Everything linked to the election, so an officer sees the whole state
    // without opening each section: the positions, who is approved and who is
    // still awaiting review.
    const [positionResult, candidateResult] = await Promise.all([
      positions.list(id).catch(() => ({ positions: [] as Position[] })),
      candidates.list(id).catch(() => ({ candidates: [] as Candidate[] })),
    ]);
    setPositionList(positionResult.positions ?? []);
    setCandidateList(candidateResult.candidates ?? []);
    setNotes(messages);
  }, [id, toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  async function run(action: () => Promise<unknown>, message: string) {
    setBusy(true);
    try {
      await action();
      toast.success(message);
      await load();
      return true;
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The change was refused.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function update(event: React.FormEvent) {
    event.preventDefault();
    setFields({});
    await run(
      () =>
        elections.update(id, {
          title: form.title,
          description: form.description,
          starts_at: toISO(form.starts_at),
          ends_at: toISO(form.ends_at),
          public_listing: form.public_listing,
          reason: form.reason,
        }),
      "Configuration saved."
    );
  }

  async function transition() {
    const ok = await run(
      () => elections.transition(id, pendingTo, reason),
      `State changed to ${statusLabel(pendingTo as Election["status"])}.`
    );
    if (ok) {
      setPendingTo("");
      setReason("");
    }
  }

  async function saveEligibility(event: React.FormEvent) {
    event.preventDefault();
    const ok = await run(
      () =>
        elections.update(id, {
          allowed_domain: eligibility.allowed_domain.trim(),
          allowed_years: eligibility.allowed_years.trim(),
          allowed_courses: eligibility.allowed_courses.trim(),
          require_roster: eligibility.require_roster,
          reason: form.reason,
        }),
      "Eligibility rule saved."
    );
    if (ok) await load();
  }

  async function cloneElection(event: React.FormEvent) {
    event.preventDefault();
    if (!cloneForm.starts_at || !cloneForm.ends_at) {
      toast.error("Choose when the copied election opens and closes.");
      return;
    }
    setCloneBusy(true);
    try {
      const copy = await elections.clone(id, {
        title: cloneForm.title.trim() || undefined,
        starts_at: toISO(cloneForm.starts_at),
        ends_at: toISO(cloneForm.ends_at),
      });
      toast.success("Election copied as a new draft.");
      router.push(`/staff/elections/${copy.id}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The election could not be copied.");
    } finally {
      setCloneBusy(false);
    }
  }

  async function importRoster(event: React.FormEvent) {
    event.preventDefault();
    const file = rosterFileRef.current?.files?.[0];
    if (!file) {
      toast.error("Choose a roster CSV file.");
      return;
    }
    setRosterBusy(true);
    try {
      const { preview } = await elections.roster.preview(id, file);
      if (preview.errors.length > 0) {
        toast.error(`${preview.errors.length} row(s) are invalid: ${preview.errors[0].reason}`);
        return;
      }
      await elections.roster.import(id, file);
      toast.success(`Roster imported: ${preview.creates} added, ${preview.updates} updated.`);
      setRosterList(await elections.roster.list(id));
      if (rosterFileRef.current) rosterFileRef.current.value = "";
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The roster could not be imported.");
    } finally {
      setRosterBusy(false);
    }
  }

  async function revokeRoster(entry: RosterEntry) {
    const reason = window.prompt(`Why should ${entry.email} lose eligibility?`);
    if (!reason?.trim()) return;
    setRosterBusy(true);
    try {
      await elections.roster.revoke(id, entry.id, reason.trim());
      setRosterList(await elections.roster.list(id));
      toast.success("Voter removed from the list.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The voter could not be removed.");
    } finally {
      setRosterBusy(false);
    }
  }

  async function assignRole(event: React.FormEvent) {
    event.preventDefault();
    if (!roleEmail.trim()) {
      toast.error("Enter the email address of the account to assign.");
      return;
    }
    setRoleBusy(true);
    try {
      const user = await elections.roles.lookup(id, roleEmail.trim());
      await elections.roles.assign(id, user.user_id, roleName);
      toast.success(
        `${user.full_name || user.email} was assigned ${roleName.replaceAll("_", " ")}.`
      );
      setRoleEmail("");
      const next = await elections.roles.list(id);
      setRoles(next);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The role could not be assigned.");
    } finally {
      setRoleBusy(false);
    }
  }

  async function removeRole(userId: string, role: string) {
    setRoleBusy(true);
    try {
      await elections.roles.remove(id, userId, role);
      setRoles(await elections.roles.list(id));
      toast.success("Role removed.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The role could not be removed.");
    } finally {
      setRoleBusy(false);
    }
  }

  async function certify() {
    const snapshotId = results?.snapshot?.id;
    if (!snapshotId) {
      toast.error("Run the tally before certifying these results.");
      return;
    }
    if (!certifyReason.trim()) {
      toast.error("Add a reason before certifying.");
      return;
    }
    const ok = await run(
      () => elections.certify(id, snapshotId, certifyReason.trim()),
      "Results certified."
    );
    if (ok) {
      setConfirmCertify(false);
      setCertifyReason("");
    }
  }

  async function removeElection() {
    setBusy(true);
    try {
      await elections.remove(id);
      toast.success("Election deleted.");
      router.replace("/staff");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The election could not be deleted.");
      setBusy(false);
      setConfirmDelete(false);
    }
  }

  if (loadError) {
    return (
      <StaffShell title="Manage election">
        <LoadError
          title="We could not open this election"
          message={loadError}
          backHref="/staff"
          backLabel="Back to elections"
        />
      </StaffShell>
    );
  }

  if (!election) {
    return (
      <StaffShell title="Manage election">
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "35%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
        <p style={{ marginTop: 16 }}>
          <Link className="btn" href="/staff">
            Back to elections
          </Link>
        </p>
      </StaffShell>
    );
  }

  // Opening is automatic at the start time, so the only manual transition the
  // three-stage machine offers is closing early.
  const transitions = (election.allowed_transitions ?? []).filter((to) => to !== "OPEN");
  const frozen = Boolean(election.eligibility_frozen_at);

  return (
    <StaffShell title={election.title}>
      <div className="page-head" style={{ marginBottom: 20 }}>
        <div>
          <div className="row" style={{ marginBottom: 6 }}>
            <StatusChip status={election.status} />
            <span className="tag">{categoryLabel(election.category)}</span>
            <span className="mono hint">v{election.version}</span>
          </div>
          <h1>{election.title}</h1>
          <p>
            {formatEAT(election.starts_at)} to {formatEAT(election.ends_at)}
          </p>
        </div>
        <Link className="btn" href={`/elections/${election.id}`}>
          <Icon name="eye" size="s" />
          Voter view
        </Link>
      </div>

      {notes.map((note) => (
        <p className="hint" key={note}>
          {note}
        </p>
      ))}

      <div className="grid2" style={{ marginBottom: 20 }}>
        <div className="panel">
          <div className="kpis" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <div className="kpi">
              <span className="l">Eligible voters</span>
              <span className="v">{formatNumber(report?.eligible ?? 0)}</span>
              <span className="s">{frozen ? "frozen roster" : "roster not frozen"}</span>
            </div>
            <div className="kpi">
              <span className="l">Turnout</span>
              <span className="v">{formatPercent(report?.turnout_percent ?? 0)}</span>
              <span className="s">of {formatNumber(report?.eligible ?? 0)} eligible</span>
            </div>
          </div>
        </div>
        <div className="panel stack">
          <span className="lbl">Eligibility</span>
          <p className="hint">
            {frozen
              ? `Frozen at ${formatEAT(election.eligibility_frozen_at ?? "")} with ${formatNumber(
                  election.eligibility_frozen_count
                )} voters.`
              : "Freezing records the eligible count when voting opens. It is idempotent."}
          </p>
          <div className="row">
            <button
              type="button"
              className="btn sm"
              disabled={busy || frozen}
              onClick={() => run(() => elections.freeze(id), "Eligibility frozen.")}
            >
              <Icon name="lock" size="s" />
              {frozen ? "Frozen" : "Freeze eligibility"}
            </button>
          </div>
        </div>
      </div>

      <section className="panel" id="eligibility" style={{ marginBottom: 20 }}>
        <div className="section-h" style={{ marginTop: 0 }}>
          <h2>Eligibility rule</h2>
          <span className="mono">
            {report ? `${formatNumber(report.eligible)} registered users match` : "count loading"}
          </span>
        </div>
        <form className="form-stack" onSubmit={saveEligibility}>
          <div className="form-grid">
            <div className="field">
              <label htmlFor="eligibility-domain">Student email domain</label>
              <input
                id="eligibility-domain"
                className="in"
                value={eligibility.allowed_domain}
                onChange={(event) =>
                  setEligibility({
                    ...eligibility,
                    allowed_domain: event.target.value,
                  })
                }
                placeholder="std.mast.ac.ug"
              />
            </div>
            <div className="field">
              <label htmlFor="eligibility-years">
                Enrolment years (comma separated, blank = any)
              </label>
              <input
                id="eligibility-years"
                className="in"
                value={eligibility.allowed_years}
                onChange={(event) =>
                  setEligibility({
                    ...eligibility,
                    allowed_years: event.target.value,
                  })
                }
                placeholder="2024"
              />
            </div>
            <div className="field">
              <label htmlFor="eligibility-courses">
                Course codes (comma separated, blank = any)
              </label>
              <input
                id="eligibility-courses"
                className="in"
                value={eligibility.allowed_courses}
                onChange={(event) =>
                  setEligibility({
                    ...eligibility,
                    allowed_courses: event.target.value,
                  })
                }
                placeholder="bse"
              />
            </div>
            <div className="field">
              <label htmlFor="eligibility-roster">
                <input
                  id="eligibility-roster"
                  type="checkbox"
                  checked={eligibility.require_roster}
                  onChange={(event) =>
                    setEligibility({
                      ...eligibility,
                      require_roster: event.target.checked,
                    })
                  }
                />{" "}
                Only voters on the imported list may see or vote
              </label>
              <span className="hint">
                The list replaces the rule above as the electorate. Add or remove names until the
                election closes; nothing changes after that.
              </span>
            </div>
            <div className="field">
              <label htmlFor="eligibility-reason">Reason (required once frozen)</label>
              <input
                id="eligibility-reason"
                className="in"
                value={form.reason}
                onChange={(event) => setForm({ ...form, reason: event.target.value })}
                placeholder="Why the audience changed"
              />
            </div>
          </div>
          <p className="hint">
            Example:{" "}
            <span className="mono">
              {(eligibility.allowed_years.split(",")[0] || "2024").trim().toUpperCase()}
              {(eligibility.allowed_courses.split(",")[0] || "BSE").trim().toUpperCase()}
              088@{eligibility.allowed_domain || "std.mast.ac.ug"}
            </span>{" "}
            is eligible. Empty year or course lists mean “any”.
          </p>
          <div>
            <button className="btn primary" type="submit" disabled={busy}>
              Save eligibility rule
            </button>
          </div>
        </form>
      </section>

      {election.require_roster ? (
        <section className="panel stack" id="roster" style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Voter list</h2>
            <span className="mono">{rosterList ? `${rosterList.length} listed` : "loading"}</span>
          </div>
          <p className="hint" style={{ marginTop: 0 }}>
            Only these emails see the election and may vote in it. The CSV needs the columns
            student_number, email and full_name. Re-importing updates rows in place; removing a
            voter revokes eligibility and keeps the row for the audit trail.
          </p>
          <form className="row" onSubmit={importRoster}>
            <input ref={rosterFileRef} type="file" accept=".csv,text/csv" className="in" />
            <button className="btn primary" type="submit" disabled={rosterBusy}>
              {rosterBusy ? "Working…" : "Import CSV"}
            </button>
          </form>
          <div className="tbl-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Voter</th>
                  <th>Student number</th>
                  <th>Status</th>
                  <th aria-label="Actions" />
                </tr>
              </thead>
              <tbody>
                {(rosterList ?? []).length === 0 ? (
                  <tr>
                    <td colSpan={4}>
                      No voters listed yet. Until the list has entries, nobody can see this
                      election.
                    </td>
                  </tr>
                ) : (
                  (rosterList ?? []).map((entry) => (
                    <tr key={entry.id}>
                      <td>
                        <b>{entry.full_name}</b>
                        <div className="hint">{entry.email}</div>
                      </td>
                      <td className="mono">{entry.student_number}</td>
                      <td>
                        <span className="chip">{entry.status.toLowerCase()}</span>
                      </td>
                      <td>
                        <div className="table-actions">
                          {entry.status === "ELIGIBLE" ? (
                            <button
                              type="button"
                              className="btn sm danger-o"
                              disabled={rosterBusy}
                              onClick={() => revokeRoster(entry)}
                            >
                              Remove
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      <div className="split">
        <div className="stack">
          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Positions &amp; candidates</h2>
              <span className="mono">
                {positionList.length} position{positionList.length === 1 ? "" : "s"}
              </span>
            </div>
            {positionList.length === 0 ? (
              <div className="stack" style={{ gap: 10 }}>
                <p className="hint" style={{ margin: 0 }}>
                  This election has no positions yet. A draft opens automatically only once every
                  position has at least one approved candidate, so add a position here and approve
                  its candidates.
                </p>
                <Link className="btn sm primary" href={`/staff/elections/${id}/positions`}>
                  <Icon name="plus" size="s" />
                  Add a position
                </Link>
              </div>
            ) : (
              <div className="stack">
                {positionList.map((position) => {
                  const applications = candidateList.filter(
                    (candidate) => candidate.position_id === position.id
                  );
                  const approved = applications.filter(
                    (candidate) => candidate.approval_status === "APPROVED"
                  );
                  const awaiting = applications.filter(
                    (candidate) =>
                      candidate.approval_status === "NOMINATED" ||
                      candidate.approval_status === "PENDING"
                  );
                  return (
                    <div
                      className="panel stack"
                      key={position.id}
                      style={{ background: "var(--sunken)" }}
                    >
                      <div className="row-between">
                        <b>{position.title}</b>
                        <span className="hint">
                          {approved.length} approved · {applications.length} applied
                        </span>
                      </div>
                      {approved.length > 0 ? (
                        <div className="stack" style={{ gap: 4 }}>
                          <span className="lbl">Approved candidates</span>
                          {approved.map((candidate) => (
                            <span key={candidate.id} className="hint">
                              {candidate.ballot_number ? `#${candidate.ballot_number} · ` : ""}
                              {candidate.full_name}
                              {candidate.manifesto || candidate.manifesto_url
                                ? " · manifesto ready"
                                : " · no manifesto yet"}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="hint">
                          No approved candidates yet — add and approve at least one candidate before
                          the election can open.
                        </span>
                      )}
                      {awaiting.length > 0 ? (
                        <div className="stack" style={{ gap: 4 }}>
                          <span className="lbl">Applicants awaiting review</span>
                          {awaiting.map((candidate) => (
                            <span key={candidate.id} className="hint">
                              {candidate.full_name}
                            </span>
                          ))}
                        </div>
                      ) : null}
                      <div className="row">
                        <Link className="btn sm" href={`/staff/elections/${id}/candidates`}>
                          Review candidates
                        </Link>
                        <Link className="btn sm" href={`/staff/elections/${id}/positions`}>
                          Edit positions
                        </Link>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </section>

          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Lifecycle</h2>
              <span className="mono">server clock</span>
            </div>
            <div className="rail" style={{ marginBottom: 16 }}>
              {["DRAFT", "OPEN", "CLOSED"].map((state) => (
                <div key={state} className={`ls${state === election.status ? " cur" : ""}`}>
                  {statusLabel(state as Election["status"])}
                </div>
              ))}
            </div>
            <p className="hint">
              A draft opens automatically at its start time and closes at its end time. An early
              close needs a reason.
            </p>
            {readiness && !readiness.ready ? (
              <div className="banner hatch" role="status" style={{ marginTop: 12 }}>
                <Icon name="alert" size="s" />
                <div>
                  <b>Not ready to open</b>
                  {Object.entries(readiness.problems).map(([field, problem]) => (
                    <span key={field} className="hint" style={{ display: "block" }}>
                      {problem}
                    </span>
                  ))}
                </div>
              </div>
            ) : null}
            <div className="row" style={{ marginTop: 14 }}>
              {transitions.length === 0 ? (
                <span className="hint">No further transitions are available.</span>
              ) : (
                transitions.map((to) => (
                  <button
                    key={to}
                    type="button"
                    className={`btn sm${to === "CLOSED" ? " danger-o" : ""}`}
                    disabled={busy}
                    onClick={() => {
                      setPendingTo(to);
                      setReason("");
                    }}
                  >
                    {transitionLabel(to)}
                  </button>
                ))
              )}
              {election.status === "CLOSED" ? (
                <button
                  type="button"
                  className="btn sm primary"
                  disabled={busy}
                  onClick={() => setConfirmCertify(true)}
                >
                  <Icon name="shield" size="s" />
                  Certify results
                </button>
              ) : null}
            </div>
          </section>

          <section className="panel" id="schedule">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Schedule</h2>
              <span className="mono">
                {frozen ? "live · title, description, extension" : "before open"}
              </span>
            </div>
            <form className="form-stack" onSubmit={update}>
              <div className="field">
                <label htmlFor="edit-title">Title</label>
                <input
                  id="edit-title"
                  className={`in${fields.title ? " bad" : ""}`}
                  value={form.title}
                  onChange={(event) => setForm({ ...form, title: event.target.value })}
                />
                {fields.title ? <span className="err">{fields.title}</span> : null}
              </div>
              <div className="field">
                <label htmlFor="edit-description">Description</label>
                <textarea
                  id="edit-description"
                  className="ta"
                  value={form.description}
                  onChange={(event) => setForm({ ...form, description: event.target.value })}
                />
              </div>
              <div className="form-grid">
                <div className="field">
                  <label htmlFor="edit-start">Voting opens (EAT)</label>
                  <input
                    id="edit-start"
                    className={`in${fields.starts_at ? " bad" : ""}`}
                    type="datetime-local"
                    value={form.starts_at}
                    disabled={frozen}
                    onChange={(event) => setForm({ ...form, starts_at: event.target.value })}
                  />
                  {fields.starts_at ? <span className="err">{fields.starts_at}</span> : null}
                </div>
                <div className="field">
                  <label htmlFor="edit-end">Voting closes (EAT)</label>
                  <input
                    id="edit-end"
                    className={`in${fields.ends_at ? " bad" : ""}`}
                    type="datetime-local"
                    value={form.ends_at}
                    onChange={(event) => setForm({ ...form, ends_at: event.target.value })}
                  />
                  {fields.ends_at ? <span className="err">{fields.ends_at}</span> : null}
                </div>
                <div className="field full">
                  <span className="lbl">Quick length from the opening time</span>
                  <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
                    {durationPresets.map((preset) => (
                      <button
                        key={preset.label}
                        type="button"
                        className="btn sm"
                        disabled={!form.starts_at}
                        onClick={() =>
                          setForm({
                            ...form,
                            ends_at: addMinutes(form.starts_at, preset.minutes),
                          })
                        }
                      >
                        {preset.label}
                      </button>
                    ))}
                  </div>
                  <span className="hint">Any custom length is allowed too.</span>
                </div>
                <div className="field full">
                  <label className="chk">
                    <input
                      type="checkbox"
                      checked={form.public_listing}
                      disabled={frozen}
                      onChange={(event) =>
                        setForm({ ...form, public_listing: event.target.checked })
                      }
                    />
                    <span>List this election on the public landing page</span>
                  </label>
                  <span className="hint">
                    {frozen
                      ? "The configuration is locked after voting opens."
                      : "Shows the title, category, dates and position count on the public page until voting opens. No voter or ballot data is exposed."}
                  </span>
                </div>
              </div>
              <div className="field">
                <label htmlFor="edit-reason">Reason for the change</label>
                <input
                  id="edit-reason"
                  className="in"
                  value={form.reason}
                  onChange={(event) => setForm({ ...form, reason: event.target.value })}
                  placeholder="Required once voting has opened."
                />
                <span className="hint">Every change is written to the election audit trail.</span>
              </div>
              <div>
                <button className="btn primary" type="submit" disabled={busy}>
                  Save configuration
                </button>
              </div>
            </form>
          </section>

          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Results</h2>
              {results ? <span className="lbl">{results.label}</span> : null}
            </div>
            {!results?.snapshot ? (
              <p className="hint">
                {results?.reason ?? "Results are not available to you for this election yet."}
              </p>
            ) : results.snapshot.positions.length === 0 ? (
              <p className="hint">No tally rows have been computed.</p>
            ) : (
              <div className="stack">
                {results.snapshot.positions.map((position) => (
                  <div key={position.position_id}>
                    <div className="row-between">
                      <b>{position.title}</b>
                      <span className="mono">
                        {position.outcome.replaceAll("_", " ").toLowerCase()}
                      </span>
                    </div>
                    <div className="tally" style={{ marginTop: 10 }}>
                      {position.rows
                        .slice()
                        .sort((a, b) => b.votes - a.votes)
                        .map((row) => (
                          <div className="trow" key={row.candidate_id}>
                            <span>
                              <b>{row.name}</b>
                              {row.is_winner ? <span className="lead-tag">Leading</span> : null}
                            </span>
                            <div className={`bar${row.is_winner ? " lead" : ""}`}>
                              <i
                                style={{
                                  width: `${Math.min(100, Math.max(0, row.share_percent))}%`,
                                }}
                              />
                            </div>
                            <span className="vl">
                              <b>{formatNumber(row.votes)}</b> · {formatPercent(row.share_percent)}
                            </span>
                          </div>
                        ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>

        <aside className="stack">
          {canManageRoles ? (
            <div className="panel stack">
              <span className="lbl">Temporary election roles</span>
              <form className="form-stack" onSubmit={assignRole}>
                <div className="field">
                  <label htmlFor="role-email">Account email</label>
                  <input
                    id="role-email"
                    className="in"
                    type="email"
                    value={roleEmail}
                    onChange={(event) => setRoleEmail(event.target.value)}
                    placeholder="organizer@std.mast.ac.ug"
                  />
                </div>
                <div className="field">
                  <label htmlFor="role-name">Role on this election</label>
                  <select
                    id="role-name"
                    className="sel"
                    value={roleName}
                    onChange={(event) => setRoleName(event.target.value)}
                  >
                    <option value="ELECTION_OFFICIAL">Election organizer</option>
                    <option value="CHIEF_ELECTION_ADMIN">Chief election admin</option>
                    <option value="ELECTION_AUDITOR">Election auditor</option>
                    <option value="CANDIDATE">Candidate</option>
                  </select>
                </div>
                <button className="btn sm primary" type="submit" disabled={roleBusy}>
                  {roleBusy ? "Working…" : "Assign role"}
                </button>
              </form>
              {roles.length === 0 ? (
                <span className="hint">No temporary roles assigned.</span>
              ) : (
                <div className="stack">
                  {roles.map((role) => (
                    <div className="row-between" key={role.user_id + role.role}>
                      <span>
                        <b>{role.full_name || role.email || role.user_id}</b>
                        <span className="hint" style={{ display: "block" }}>
                          {role.email} · {role.role.replaceAll("_", " ").toLowerCase()}
                        </span>
                      </span>
                      <button
                        type="button"
                        className="btn sm danger-o"
                        disabled={roleBusy}
                        onClick={() => removeRole(role.user_id, role.role)}
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : null}

          <div className="panel stack">
            <span className="lbl">Re-run this election</span>
            <p className="hint" style={{ marginTop: 0 }}>
              Copies the rules, positions, candidates and voter list into a new draft. Votes,
              results, ballot versions and role assignments stay behind, and the copy is yours.
            </p>
            <form className="form-stack" onSubmit={cloneElection}>
              <div className="field">
                <label htmlFor="clone-title">Title (blank keeps the current one)</label>
                <input
                  id="clone-title"
                  className="in"
                  value={cloneForm.title}
                  onChange={(event) => setCloneForm({ ...cloneForm, title: event.target.value })}
                  placeholder={`${election.title} (copy)`}
                />
              </div>
              <div className="field">
                <label htmlFor="clone-start">Voting opens (EAT)</label>
                <input
                  id="clone-start"
                  className="in"
                  type="datetime-local"
                  value={cloneForm.starts_at}
                  onChange={(event) =>
                    setCloneForm({ ...cloneForm, starts_at: event.target.value })
                  }
                />
              </div>
              <div className="field">
                <label htmlFor="clone-end">Voting closes (EAT)</label>
                <input
                  id="clone-end"
                  className="in"
                  type="datetime-local"
                  value={cloneForm.ends_at}
                  onChange={(event) => setCloneForm({ ...cloneForm, ends_at: event.target.value })}
                />
              </div>
              <button className="btn sm" type="submit" disabled={cloneBusy}>
                <Icon name="copy" size="s" />
                {cloneBusy ? "Copying…" : "Copy as a new draft"}
              </button>
            </form>
          </div>

          <div className="panel stack">
            <span className="lbl">Audit</span>
            <dl className="kv">
              <dt>Created</dt>
              <dd>{formatEAT(election.created_at)}</dd>
              <dt>Last version</dt>
              <dd>v{election.version}</dd>
              <dt>Result policy</dt>
              <dd>{election.result_policy.replaceAll("_", " ").toLowerCase()}</dd>
              <dt>Accepting votes</dt>
              <dd>{election.accepting_votes ? "Yes" : "No"}</dd>
            </dl>
            <button
              type="button"
              className="btn sm danger-o"
              disabled={busy}
              onClick={() => setConfirmDelete(true)}
            >
              <Icon name="trash" size="s" />
              Delete election
            </button>
          </div>
          <div className="panel stack">
            <span className="lbl">Roster</span>
            {report ? (
              <dl className="kv">
                <dt>Total rows</dt>
                <dd>{formatNumber(report.total)}</dd>
                <dt>Eligible</dt>
                <dd>{formatNumber(report.eligible)}</dd>
                <dt>Participated</dt>
                <dd>{formatNumber(report.participated)}</dd>
                <dt>Last import</dt>
                <dd>{report.last_import_at ? formatEAT(report.last_import_at) : "—"}</dd>
              </dl>
            ) : (
              <span className="hint">No roster report available.</span>
            )}
          </div>
        </aside>
      </div>

      <ConfirmDialog
        open={pendingTo !== ""}
        title={`Move to ${statusLabel(pendingTo as Election["status"])}?`}
        confirmLabel="Confirm change"
        destructive={pendingTo === "CLOSED"}
        busy={busy}
        onClose={() => setPendingTo("")}
        onConfirm={transition}
      >
        <p>
          The election will move from <b>{statusLabel(election.status)}</b> to{" "}
          <b>{statusLabel(pendingTo as Election["status"])}</b>. The change is audited.
        </p>
        <div className="field">
          <label htmlFor="transition-reason">
            Reason {pendingTo === "CLOSED" ? "(required to close early)" : "(optional)"}
          </label>
          <textarea
            id="transition-reason"
            className="ta"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={confirmCertify}
        title="Certify these results?"
        confirmLabel="Certify"
        busy={busy}
        onClose={() => setConfirmCertify(false)}
        onConfirm={certify}
      >
        <p>
          Certification finalises the result. The election creator cannot certify, and the change is
          recorded in the audit trail.
        </p>
        <div className="field">
          <label htmlFor="certify-reason">Reason (required)</label>
          <textarea
            id="certify-reason"
            className="ta"
            value={certifyReason}
            onChange={(event) => setCertifyReason(event.target.value)}
            placeholder="Why are these results being certified?"
          />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={confirmDelete}
        title="Delete this election?"
        confirmLabel="Delete election"
        destructive
        busy={busy}
        onClose={() => setConfirmDelete(false)}
        onConfirm={removeElection}
      >
        <p>
          This permanently removes <b>{election.title}</b> together with its positions and
          candidates. An election that has already collected a ballot cannot be deleted — close it
          instead so its result stays auditable.
        </p>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function ManageElectionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth staffOnly>
      <Manage id={id} />
    </RequireAuth>
  );
}
