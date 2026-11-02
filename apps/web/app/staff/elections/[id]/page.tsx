"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ElectionTabs } from "@/components/election-tabs";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import {
  EAT,
  categoryLabel,
  formatEAT,
  formatNumber,
  formatPercent,
  statusLabel,
} from "@/lib/format";
import type { Election, ElectionRole, EligibilityReport, Results } from "@/lib/types";

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

function transitionLabel(to: string): string {
  switch (to) {
    case "REGISTRATION":
      return "Open applications";
    case "SCHEDULED":
      return "Schedule election";
    case "PENDING_APPROVAL":
      return "Send for approval";
    case "CLOSED":
      return "Close early";
    case "RESULTS_UNDER_REVIEW":
      return "Review results";
    case "ARCHIVED":
      return "Archive";
    case "CANCELLED":
      return "Cancel";
    default:
      return statusLabel(to as Election["status"]);
  }
}

function Manage({ id }: { id: string }) {
  const [election, setElection] = useState<Election | null>(null);
  const [report, setReport] = useState<EligibilityReport | null>(null);
  const [results, setResults] = useState<Results | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [notes, setNotes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [pendingTo, setPendingTo] = useState("");
  const [reason, setReason] = useState("");
  const [confirmCertify, setConfirmCertify] = useState(false);
  const [certifyReason, setCertifyReason] = useState("");
  const [form, setForm] = useState({ title: "", starts_at: "", ends_at: "", reason: "" });
  const [fields, setFields] = useState<Record<string, string>>({});
  const [eligibility, setEligibility] = useState({
    allowed_domain: "",
    allowed_years: "",
    allowed_courses: "",
  });
  const [readiness, setReadiness] = useState<{
    ready: boolean;
    problems: Record<string, string>;
  } | null>(null);
  const [roles, setRoles] = useState<ElectionRole[]>([]);
  const [roleEmail, setRoleEmail] = useState("");
  const [roleName, setRoleName] = useState("ELECTION_OFFICIAL");
  const [roleBusy, setRoleBusy] = useState(false);

  const load = useCallback(async () => {
    const messages: string[] = [];
    try {
      const current = await elections.get(id);
      setElection(current);
      setForm({
        title: current.title,
        starts_at: toEatInput(current.starts_at),
        ends_at: toEatInput(current.ends_at),
        reason: "",
      });
      setEligibility({
        allowed_domain: current.allowed_domain ?? "",
        allowed_years: current.allowed_years ?? "",
        allowed_courses: current.allowed_courses ?? "",
      });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Election could not be loaded.");
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
    setNotes(messages);
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  async function run(action: () => Promise<unknown>, message: string) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await action();
      setNotice(message);
      await load();
      return true;
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The change was refused.");
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
          starts_at: toISO(form.starts_at),
          ends_at: toISO(form.ends_at),
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
          reason: form.reason,
        }),
      "Eligibility rule saved."
    );
    if (ok) await load();
  }

  async function assignRole(event: React.FormEvent) {
    event.preventDefault();
    if (!roleEmail.trim()) {
      setError("Enter the email address of the account to assign.");
      return;
    }
    setRoleBusy(true);
    setError("");
    setNotice("");
    try {
      const user = await elections.roles.lookup(id, roleEmail.trim());
      await elections.roles.assign(id, user.user_id, roleName);
      setNotice(`${user.full_name || user.email} was assigned ${roleName.replaceAll("_", " ")}.`);
      setRoleEmail("");
      const next = await elections.roles.list(id);
      setRoles(next);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The role could not be assigned.");
    } finally {
      setRoleBusy(false);
    }
  }

  async function removeRole(userId: string, role: string) {
    setRoleBusy(true);
    setError("");
    setNotice("");
    try {
      await elections.roles.remove(id, userId, role);
      setRoles(await elections.roles.list(id));
      setNotice("Role removed.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The role could not be removed.");
    } finally {
      setRoleBusy(false);
    }
  }

  async function certify() {
    const snapshotId = results?.snapshot?.id;
    if (!snapshotId) {
      setError("Run the tally before certifying these results.");
      return;
    }
    if (!certifyReason.trim()) {
      setError("Add a reason before certifying.");
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

  if (!election) {
    return (
      <StaffShell title="Manage election">
        <div className="panel stack" aria-busy="true">
          {error ? (
            <div className="banner bad" role="alert">
              <Icon name="alert" />
              <div>
                <b>Election unavailable</b>
                {error}
              </div>
            </div>
          ) : (
            <>
              <span className="skel" style={{ height: 18, width: "35%" }} />
              <span className="skel" style={{ height: 14 }} />
            </>
          )}
        </div>
        <p style={{ marginTop: 16 }}>
          <Link className="btn" href="/staff">
            Back to elections
          </Link>
        </p>
      </StaffShell>
    );
  }

  const transitions = (election.allowed_transitions ?? []).filter(
    (to) => to !== "OPEN" && to !== "FINALIZED"
  );
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

      <ElectionTabs id={election.id} />

      {error ? (
        <div className="banner bad" role="alert" style={{ marginBottom: 16 }}>
          <Icon name="alert" />
          <div>
            <b>Request refused</b>
            {error}
          </div>
        </div>
      ) : null}
      {notice ? (
        <div className="banner ok" role="status" style={{ marginBottom: 16 }}>
          <Icon name="check" />
          <div>{notice}</div>
        </div>
      ) : null}
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
                  setEligibility({ ...eligibility, allowed_domain: event.target.value })
                }
                placeholder="std.mast.ac.ug"
              />
            </div>
            <div className="field">
              <label htmlFor="eligibility-years">Enrolment years (comma separated, blank = any)</label>
              <input
                id="eligibility-years"
                className="in"
                value={eligibility.allowed_years}
                onChange={(event) =>
                  setEligibility({ ...eligibility, allowed_years: event.target.value })
                }
                placeholder="2024"
              />
            </div>
            <div className="field">
              <label htmlFor="eligibility-courses">Course codes (comma separated, blank = any)</label>
              <input
                id="eligibility-courses"
                className="in"
                value={eligibility.allowed_courses}
                onChange={(event) =>
                  setEligibility({ ...eligibility, allowed_courses: event.target.value })
                }
                placeholder="bse"
              />
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

      <div className="split">
        <div className="stack">
          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Lifecycle</h2>
              <span className="mono">server clock</span>
            </div>
            <div className="rail" style={{ marginBottom: 16 }}>
              {[
                "DRAFT",
                "REGISTRATION",
                "PENDING_APPROVAL",
                "SCHEDULED",
                "OPEN",
                "CLOSED",
                "RESULTS_UNDER_REVIEW",
                "FINALIZED",
                "ARCHIVED",
              ].map((state) => (
                <div key={state} className={`ls${state === election.status ? " cur" : ""}`}>
                  {state === "PENDING_APPROVAL"
                    ? "Approval"
                    : state === "RESULTS_UNDER_REVIEW"
                      ? "Review"
                      : statusLabel(state as Election["status"])}
                </div>
              ))}
            </div>
            <p className="hint">
              Opening and closing follow the schedule automatically. Cancel, archive and an early
              close need a reason and the publisher permission.
            </p>
            {readiness && !readiness.ready ? (
              <div className="banner hatch" role="status" style={{ marginTop: 12 }}>
                <Icon name="alert" size="s" />
                <div>
                  <b>Not ready to schedule</b>
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
                    className={`btn sm${to === "CANCELLED" ? " danger-o" : ""}`}
                    disabled={busy || (to === "SCHEDULED" && readiness !== null && !readiness.ready)}
                    onClick={() => {
                      setPendingTo(to);
                      setReason("");
                    }}
                  >
                    {transitionLabel(to)}
                  </button>
                ))
              )}
              {["CLOSED", "RESULTS_UNDER_REVIEW"].includes(election.status) ? (
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
              <span className="mono">{frozen ? "frozen · reason required" : "before open"}</span>
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
              <div className="form-grid">
                <div className="field">
                  <label htmlFor="edit-start">Voting opens (EAT)</label>
                  <input
                    id="edit-start"
                    className={`in${fields.starts_at ? " bad" : ""}`}
                    type="datetime-local"
                    value={form.starts_at}
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
          <div className="panel stack">
            <span className="lbl">Quick Actions</span>
            <div className="stack">
              <Link className="btn sm" href={`/staff/elections/${id}/positions`}>
                <Icon name="list" size="s" />
                Manage Positions
              </Link>
              <Link className="btn sm" href={`/staff/elections/${id}/candidates`}>
                <Icon name="users" size="s" />
                Manage Candidates
              </Link>
              <Link className="btn sm" href={`/staff/elections/${id}/ballot`}>
                <Icon name="file" size="s" />
                Build Ballot
              </Link>
              <Link className="btn sm" href={`/staff/elections/${id}/monitor`}>
                <Icon name="chart" size="s" />
                Live Monitor
              </Link>
              <Link className="btn sm" href={`/staff/elections/${id}/turnout`}>
                <Icon name="activity" size="s" />
                Turnout
              </Link>
            </div>
          </div>
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
        destructive={pendingTo === "CANCELLED"}
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
            Reason{" "}
            {pendingTo === "CANCELLED" || pendingTo === "ARCHIVED" ? "(required)" : "(optional)"}
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
    </StaffShell>
  );
}

export default function ManageElectionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <Manage id={id} />
    </RequireAuth>
  );
}
