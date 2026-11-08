"use client";

import type { LandingAdminState, LandingDraft, LandingIconKey } from "@campvote/shared/types/index";
import { useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Icon } from "@/components/icon";
import { LandingPage } from "@/components/landing/landing-page";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, landing } from "@/lib/api";
import { LANDING_ICONS, LANDING_LIMITS, LANDING_MAX, visibleDocument } from "@/lib/landing";
import "../../landing.css";

const EAT_OFFSET = "+03:00";

/** ISO timestamp → the wall clock an EAT datetime-local input expects. */
function toEatInput(iso: string | null | undefined): string {
  if (!iso) return "";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("sv-SE", {
    timeZone: "Africa/Kampala",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
    .format(date)
    .replace(" ", "T");
}

function toISO(value: string): string | null {
  return value ? `${value}:00${EAT_OFFSET}` : null;
}

function stamp(iso: string | null | undefined): string {
  if (!iso) return "never";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "never";
  return new Intl.DateTimeFormat("en-GB", {
    timeZone: "Africa/Kampala",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);
}

function move<T>(list: T[], index: number, direction: -1 | 1): T[] {
  const target = index + direction;
  if (target < 0 || target >= list.length) return list;
  const next = [...list];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

function Field({
  label,
  error,
  hint,
  children,
}: {
  label: string;
  error?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="field">
      <span className="lbl">{label}</span>
      {children}
      {error ? <span className="err">{error}</span> : null}
      {hint && !error ? <span className="hint">{hint}</span> : null}
    </div>
  );
}

function Reorder({
  index,
  length,
  onMove,
}: {
  index: number;
  length: number;
  onMove: (direction: -1 | 1) => void;
}) {
  return (
    <span className="row" style={{ gap: 4 }}>
      <button
        type="button"
        className="btn ghost sm"
        aria-label="Move up"
        disabled={index === 0}
        onClick={() => onMove(-1)}
      >
        ↑
      </button>
      <button
        type="button"
        className="btn ghost sm"
        aria-label="Move down"
        disabled={index === length - 1}
        onClick={() => onMove(1)}
      >
        ↓
      </button>
    </span>
  );
}

function Editor() {
  const toast = useToast();
  const [loadError, setLoadError] = useState<string | null>(null);
  const [state, setState] = useState<LandingAdminState | null>(null);
  const [draft, setDraft] = useState<LandingDraft | null>(null);
  const [baseline, setBaseline] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [confirmRevert, setConfirmRevert] = useState(false);
  const [preview, setPreview] = useState(false);

  const apply = useCallback((next: LandingAdminState) => {
    setState(next);
    setDraft(next.draft);
    setBaseline(JSON.stringify(next.draft));
  }, []);

  const load = useCallback(async () => {
    try {
      apply(await landing.get());
      setLoadError(null);
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : "The landing page could not be loaded.");
    }
  }, [apply]);

  useEffect(() => {
    load();
  }, [load]);

  const unsaved = Boolean(draft) && JSON.stringify(draft) !== baseline;

  async function save(): Promise<boolean> {
    if (!draft) return false;
    setBusy(true);
    setFields({});
    try {
      const next = await landing.save(draft);
      apply(next);
      toast.success("Draft saved.");
      return true;
    } catch (err) {
      if (err instanceof ApiError) {
        setFields(err.fields ?? {});
        toast.error(err.message);
      } else {
        toast.error("The draft could not be saved.");
      }
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function publish() {
    if (!draft) return;
    setBusy(true);
    setFields({});
    try {
      // Publish what is on screen, not an older saved draft.
      const saved = await landing.save(draft);
      apply(saved);
      await landing.publish();
      await load();
      toast.success("Landing page published.");
    } catch (err) {
      if (err instanceof ApiError) {
        setFields(err.fields ?? {});
        toast.error(err.message);
      } else {
        toast.error("The landing page could not be published.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function revert() {
    setBusy(true);
    try {
      apply(await landing.revert());
      setConfirmRevert(false);
      toast.success("Draft reverted to the published version.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The draft could not be reverted.");
    } finally {
      setBusy(false);
    }
  }

  if (loadError) {
    return (
      <StaffShell title="Landing page">
        <LoadError
          title="We could not open the landing page editor"
          message={loadError}
          backHref="/staff"
          backLabel="Back to the dashboard"
        />
      </StaffShell>
    );
  }

  if (!draft || !state) {
    return (
      <StaffShell title="Landing page">
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "35%" }} />
          <span className="skel" style={{ height: 14 }} />
          <span className="skel" style={{ height: 14, width: "70%" }} />
        </div>
      </StaffShell>
    );
  }

  const setHero = (patch: Partial<LandingDraft["hero"]>) =>
    setDraft({ ...draft, hero: { ...draft.hero, ...patch } });
  const setBanner = (patch: Partial<LandingDraft["banner"]>) =>
    setDraft({ ...draft, banner: { ...draft.banner, ...patch } });

  return (
    <StaffShell title="Landing page">
      {preview ? (
        <div
          className="landing-admin-preview"
          role="dialog"
          aria-label="Landing page draft preview"
        >
          <div className="landing-admin-preview-bar">
            <b>Preview · draft</b>
            <button type="button" className="btn sm" onClick={() => setPreview(false)}>
              <Icon name="x" size="s" />
              Close preview
            </button>
          </div>
          <LandingPage
            document={visibleDocument(draft)}
            upcoming={[]}
            upcomingFailed={false}
            preview
          />
        </div>
      ) : null}

      <div className="page-head">
        <div>
          <span className="lbl">Public site</span>
          <h1>Landing page</h1>
          <p>Edit the public page’s words. Nothing changes for visitors until you publish.</p>
        </div>
        <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
          {unsaved ? <span className="chip review">Unsaved edits</span> : null}
          <span className={`chip ${state.dirty ? "sched" : "final"}`}>
            {state.dirty ? "Unpublished changes" : "Published"}
          </span>
        </div>
      </div>

      <div className="panel stack" style={{ marginBottom: 20 }}>
        <div className="row-between" style={{ flexWrap: "wrap", gap: 12 }}>
          <div>
            <b>Draft</b>
            <div className="hint">
              Last saved {stamp(state.updated_at)} · Published {stamp(state.published_at)} · EAT
            </div>
          </div>
          <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
            <button type="button" className="btn" disabled={busy} onClick={() => setPreview(true)}>
              <Icon name="eye" size="s" />
              Preview
            </button>
            <button
              type="button"
              className="btn"
              disabled={busy || !state.has_published}
              onClick={() => setConfirmRevert(true)}
            >
              Revert to published
            </button>
            <button type="button" className="btn primary" disabled={busy} onClick={save}>
              {unsaved ? "Save draft" : "Saved"}
            </button>
            <button type="button" className="btn primary" disabled={busy} onClick={publish}>
              Publish
            </button>
          </div>
        </div>
        <span className="hint">
          The live page picks up a publish within a minute. Security copy and illustrations are
          fixed in code and cannot be edited here.
        </span>
      </div>

      <div className="stack">
        <section className="panel stack">
          <h2>Hero</h2>
          <div className="form-stack">
            <Field
              label="Headline"
              error={fields["hero.headline"]}
              hint="Line breaks are kept. Wrap one word in *asterisks* to show it in green."
            >
              <textarea
                className="in"
                rows={3}
                maxLength={LANDING_MAX.headline}
                value={draft.hero.headline}
                onChange={(event) => setHero({ headline: event.target.value })}
              />
            </Field>
            <Field label="Supporting text" error={fields["hero.subtext"]}>
              <textarea
                className="in"
                rows={3}
                maxLength={LANDING_MAX.subtext}
                value={draft.hero.subtext}
                onChange={(event) => setHero({ subtext: event.target.value })}
              />
            </Field>
            <div className="form-grid">
              <Field label="Primary button" error={fields["hero.primary_label"]}>
                <input
                  className="in"
                  maxLength={LANDING_MAX.label}
                  value={draft.hero.primary_label}
                  onChange={(event) => setHero({ primary_label: event.target.value })}
                />
              </Field>
              <Field label="Secondary button" error={fields["hero.secondary_label"]}>
                <input
                  className="in"
                  maxLength={LANDING_MAX.label}
                  value={draft.hero.secondary_label}
                  onChange={(event) => setHero({ secondary_label: event.target.value })}
                />
              </Field>
            </div>
          </div>
        </section>

        <section className="panel stack">
          <h2>Announcement banner</h2>
          <div className="form-stack">
            <label className="chk">
              <input
                type="checkbox"
                checked={draft.banner.enabled}
                onChange={(event) => setBanner({ enabled: event.target.checked })}
              />
              <span>Show the banner above the navigation</span>
            </label>
            <Field label="Text" error={fields["banner.text"]}>
              <input
                className="in"
                maxLength={LANDING_MAX.bannerText}
                value={draft.banner.text}
                onChange={(event) => setBanner({ text: event.target.value })}
              />
            </Field>
            <Field
              label="Link"
              error={fields["banner.link"]}
              hint="A site path starting with / or an https:// address. Leave empty for plain text."
            >
              <input
                className="in"
                maxLength={LANDING_MAX.link}
                value={draft.banner.link}
                onChange={(event) => setBanner({ link: event.target.value })}
              />
            </Field>
            <div className="form-grid">
              <Field label="Show from (EAT, optional)" error={fields["banner.starts_at"]}>
                <input
                  className="in"
                  type="datetime-local"
                  value={toEatInput(draft.banner.starts_at)}
                  onChange={(event) => setBanner({ starts_at: toISO(event.target.value) })}
                />
              </Field>
              <Field label="Show until (EAT, optional)" error={fields["banner.ends_at"]}>
                <input
                  className="in"
                  type="datetime-local"
                  value={toEatInput(draft.banner.ends_at)}
                  onChange={(event) => setBanner({ ends_at: toISO(event.target.value) })}
                />
              </Field>
            </div>
          </div>
        </section>

        <section className="panel stack">
          <h2>Ticker</h2>
          <Field
            label="Phrases"
            error={fields.ticker}
            hint={`One phrase per line, up to ${LANDING_LIMITS.ticker}. They repeat in the strip under the hero.`}
          >
            <textarea
              className="in"
              rows={4}
              value={draft.ticker.join("\n")}
              onChange={(event) =>
                setDraft({
                  ...draft,
                  ticker: event.target.value.split("\n").filter((line) => line.trim() !== ""),
                })
              }
            />
          </Field>
        </section>

        <section className="panel stack">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>How voting works</h2>
            <span className="mono">{draft.how.steps.length} steps</span>
          </div>
          {fields["how.steps"] ? <span className="err">{fields["how.steps"]}</span> : null}
          {draft.how.steps.map((step, index) => (
            <div className="panel stack" key={index}>
              <div className="row-between">
                <b>Step {index + 1}</b>
                <div className="row" style={{ gap: 4 }}>
                  <Reorder
                    index={index}
                    length={draft.how.steps.length}
                    onMove={(direction) =>
                      setDraft({
                        ...draft,
                        how: { steps: move(draft.how.steps, index, direction) },
                      })
                    }
                  />
                  <button
                    type="button"
                    className="btn ghost sm"
                    onClick={() =>
                      setDraft({
                        ...draft,
                        how: {
                          steps: draft.how.steps.filter((_, i) => i !== index),
                        },
                      })
                    }
                  >
                    Remove
                  </button>
                </div>
              </div>
              <div className="form-grid">
                <Field label="Title" error={fields[`how.steps.${index}.title`]}>
                  <input
                    className="in"
                    maxLength={LANDING_MAX.stepTitle}
                    value={step.title}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        how: {
                          steps: draft.how.steps.map((s, i) =>
                            i === index ? { ...s, title: event.target.value } : s
                          ),
                        },
                      })
                    }
                  />
                </Field>
                <Field label="Text" error={fields[`how.steps.${index}.text`]}>
                  <input
                    className="in"
                    maxLength={LANDING_MAX.stepText}
                    value={step.text}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        how: {
                          steps: draft.how.steps.map((s, i) =>
                            i === index ? { ...s, text: event.target.value } : s
                          ),
                        },
                      })
                    }
                  />
                </Field>
              </div>
            </div>
          ))}
          <button
            type="button"
            className="btn"
            disabled={draft.how.steps.length >= LANDING_LIMITS.steps}
            onClick={() =>
              setDraft({
                ...draft,
                how: { steps: [...draft.how.steps, { title: "", text: "" }] },
              })
            }
          >
            <Icon name="plus" size="s" />
            Add step
          </button>
        </section>

        <section className="panel stack">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Features</h2>
            <span className="mono">{draft.features.length} cards</span>
          </div>
          {fields.features ? <span className="err">{fields.features}</span> : null}
          {draft.features.map((feature, index) => (
            <div className="panel stack" key={feature.id || index}>
              <div className="row-between">
                <label className="chk">
                  <input
                    type="checkbox"
                    checked={feature.visible}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        features: draft.features.map((f, i) =>
                          i === index ? { ...f, visible: event.target.checked } : f
                        ),
                      })
                    }
                  />
                  <span>Visible</span>
                </label>
                <div className="row" style={{ gap: 4 }}>
                  <Reorder
                    index={index}
                    length={draft.features.length}
                    onMove={(direction) =>
                      setDraft({
                        ...draft,
                        features: move(draft.features, index, direction),
                      })
                    }
                  />
                  <button
                    type="button"
                    className="btn ghost sm"
                    onClick={() =>
                      setDraft({
                        ...draft,
                        features: draft.features.filter((_, i) => i !== index),
                      })
                    }
                  >
                    Remove
                  </button>
                </div>
              </div>
              <div className="form-grid">
                <Field label="Illustration" error={fields[`features.${index}.icon`]}>
                  <select
                    className="in"
                    value={feature.icon}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        features: draft.features.map((f, i) =>
                          i === index
                            ? {
                                ...f,
                                icon: event.target.value as LandingIconKey,
                              }
                            : f
                        ),
                      })
                    }
                  >
                    {LANDING_ICONS.map((icon) => (
                      <option key={icon.key} value={icon.key}>
                        {icon.label}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="Title" error={fields[`features.${index}.title`]}>
                  <input
                    className="in"
                    maxLength={LANDING_MAX.featureTitle}
                    value={feature.title}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        features: draft.features.map((f, i) =>
                          i === index ? { ...f, title: event.target.value } : f
                        ),
                      })
                    }
                  />
                </Field>
                <div className="field full">
                  <span className="lbl">Text</span>
                  <textarea
                    className="in"
                    rows={2}
                    maxLength={LANDING_MAX.featureText}
                    value={feature.text}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        features: draft.features.map((f, i) =>
                          i === index ? { ...f, text: event.target.value } : f
                        ),
                      })
                    }
                  />
                  {fields[`features.${index}.text`] ? (
                    <span className="err">{fields[`features.${index}.text`]}</span>
                  ) : null}
                </div>
              </div>
            </div>
          ))}
          <button
            type="button"
            className="btn"
            disabled={draft.features.length >= LANDING_LIMITS.features}
            onClick={() =>
              setDraft({
                ...draft,
                features: [
                  ...draft.features,
                  {
                    id: "",
                    icon: "elections",
                    title: "",
                    text: "",
                    visible: true,
                  },
                ],
              })
            }
          >
            <Icon name="plus" size="s" />
            Add feature
          </button>
        </section>

        <section className="panel stack">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Questions</h2>
            <span className="mono">{draft.faqs.length} items</span>
          </div>
          {fields.faqs ? <span className="err">{fields.faqs}</span> : null}
          {draft.faqs.map((faq, index) => (
            <div className="panel stack" key={faq.id || index}>
              <div className="row-between">
                <label className="chk">
                  <input
                    type="checkbox"
                    checked={faq.visible}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        faqs: draft.faqs.map((q, i) =>
                          i === index ? { ...q, visible: event.target.checked } : q
                        ),
                      })
                    }
                  />
                  <span>Visible</span>
                </label>
                <div className="row" style={{ gap: 4 }}>
                  <Reorder
                    index={index}
                    length={draft.faqs.length}
                    onMove={(direction) =>
                      setDraft({
                        ...draft,
                        faqs: move(draft.faqs, index, direction),
                      })
                    }
                  />
                  <button
                    type="button"
                    className="btn ghost sm"
                    onClick={() =>
                      setDraft({
                        ...draft,
                        faqs: draft.faqs.filter((_, i) => i !== index),
                      })
                    }
                  >
                    Remove
                  </button>
                </div>
              </div>
              <Field label="Question" error={fields[`faqs.${index}.question`]}>
                <input
                  className="in"
                  maxLength={LANDING_MAX.question}
                  value={faq.question}
                  onChange={(event) =>
                    setDraft({
                      ...draft,
                      faqs: draft.faqs.map((q, i) =>
                        i === index ? { ...q, question: event.target.value } : q
                      ),
                    })
                  }
                />
              </Field>
              <Field label="Answer" error={fields[`faqs.${index}.answer`]}>
                <textarea
                  className="in"
                  rows={3}
                  maxLength={LANDING_MAX.answer}
                  value={faq.answer}
                  onChange={(event) =>
                    setDraft({
                      ...draft,
                      faqs: draft.faqs.map((q, i) =>
                        i === index ? { ...q, answer: event.target.value } : q
                      ),
                    })
                  }
                />
              </Field>
            </div>
          ))}
          <button
            type="button"
            className="btn"
            disabled={draft.faqs.length >= LANDING_LIMITS.faqs}
            onClick={() =>
              setDraft({
                ...draft,
                faqs: [...draft.faqs, { id: "", question: "", answer: "", visible: true }],
              })
            }
          >
            <Icon name="plus" size="s" />
            Add question
          </button>
        </section>

        <section className="panel stack">
          <h2>Footer</h2>
          <div className="form-stack">
            <Field label="Closing line" error={fields["footer.text"]}>
              <input
                className="in"
                maxLength={LANDING_MAX.footerText}
                value={draft.footer.text}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    footer: { ...draft.footer, text: event.target.value },
                  })
                }
              />
            </Field>
            <Field
              label="Support email (optional)"
              error={fields["footer.support_email"]}
              hint="Shown as a mailto link in the footer."
            >
              <input
                className="in"
                type="email"
                value={draft.footer.support_email}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    footer: {
                      ...draft.footer,
                      support_email: event.target.value,
                    },
                  })
                }
              />
            </Field>
          </div>
        </section>
      </div>

      <ConfirmDialog
        open={confirmRevert}
        title="Revert the draft?"
        confirmLabel="Revert draft"
        busy={busy}
        destructive
        onConfirm={revert}
        onClose={() => setConfirmRevert(false)}
      >
        <p>
          The draft returns to the currently published version. Unpublished edits cannot be
          recovered.
        </p>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function LandingAdminPage() {
  return (
    <RequireAuth adminOnly>
      <Editor />
    </RequireAuth>
  );
}
