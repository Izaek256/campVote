"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { useToast } from "@/components/toast";
import { ApiError, candidates, upload } from "@/lib/api";
import { STUDENT_EMAIL_DOMAIN } from "@/lib/config";
import { formatEAT } from "@/lib/format";
import type { CandidateApplicationView } from "@/lib/types";

function Apply({ id }: { id: string }) {
  const toast = useToast();
  const [view, setView] = useState<CandidateApplicationView | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);

  const [positionId, setPositionId] = useState("");
  const [fullName, setFullName] = useState("");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [sex, setSex] = useState("");
  const [religion, setReligion] = useState("");
  const [slogan, setSlogan] = useState("");
  const [bio, setBio] = useState("");
  const [manifesto, setManifesto] = useState("");
  const [contactEmail, setContactEmail] = useState("");
  const [contactPhone, setContactPhone] = useState("");
  const [photoUrl, setPhotoUrl] = useState("");
  const [manifestoUrl, setManifestoUrl] = useState("");
  const [passportPhotoUrl, setPassportPhotoUrl] = useState("");
  const [idDocumentUrl, setIdDocumentUrl] = useState("");
  const [transcriptUrl, setTranscriptUrl] = useState("");
  const [uploading, setUploading] = useState<
    "photo" | "manifesto" | "passport" | "id" | "transcript" | ""
  >("");

  useEffect(() => {
    candidates
      .application(id)
      .then((data) => {
        setView(data);
        setPositionId(data.positions[0]?.id ?? "");
      })
      .catch((err) => {
        const message =
          err instanceof ApiError ? err.message : "This election could not be loaded.";
        setLoadError(message);
        toast.error(message);
      });
  }, [id, toast]);

  async function pickFile(
    kind: "photo" | "manifesto" | "passport" | "id" | "transcript",
    file: File | undefined
  ) {
    if (!file) return;
    setUploading(kind);
    try {
      const isPdf = kind === "manifesto" || kind === "transcript";
      const ref = await upload(file, {
        accepts: isPdf ? "pdf" : "image",
        maxSize: (isPdf ? 5 : 2) * 1024 * 1024,
      });
      if (kind === "photo") setPhotoUrl(ref.url);
      else if (kind === "manifesto") setManifestoUrl(ref.url);
      else if (kind === "passport") setPassportPhotoUrl(ref.url);
      else if (kind === "transcript") setTranscriptUrl(ref.url);
      else setIdDocumentUrl(ref.url);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "That document could not be uploaded.");
    } finally {
      setUploading("");
    }
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!positionId) {
      toast.error("Choose the position you are applying for.");
      return;
    }
    if (fullName.trim().length < 2) {
      toast.error("Enter the name that should appear on the ballot.");
      return;
    }
    if (!dateOfBirth) {
      toast.error("Enter your date of birth.");
      return;
    }
    if (sex !== "MALE" && sex !== "FEMALE") {
      toast.error("Select your sex.");
      return;
    }
    if (!transcriptUrl) {
      toast.error("Upload your partial transcript.");
      return;
    }
    if (!passportPhotoUrl) {
      toast.error("Upload your passport photo.");
      return;
    }
    if (!idDocumentUrl) {
      toast.error("Upload a photo of your student ID.");
      return;
    }
    setBusy(true);
    try {
      await candidates.apply(id, {
        position_id: positionId,
        full_name: fullName.trim(),
        date_of_birth: dateOfBirth,
        sex,
        religion: religion.trim(),
        slogan: slogan.trim(),
        bio: bio.trim(),
        manifesto: manifesto.trim(),
        manifesto_url: manifestoUrl,
        photo_url: photoUrl,
        passport_photo_url: passportPhotoUrl,
        id_document_url: idDocumentUrl,
        partial_transcript_url: transcriptUrl,
        contact_email: contactEmail.trim(),
        contact_phone: contactPhone.trim(),
      });
      setDone(true);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Your application could not be submitted."
      );
    } finally {
      setBusy(false);
    }
  }

  if (loadError) {
    return (
      <main className="app-wrap narrow">
        <LoadError
          title="We could not open this application"
          message={loadError}
          backHref="/elections"
          backLabel="Back to elections"
        />
      </main>
    );
  }

  if (!view) {
    return (
      <main className="app-wrap narrow" aria-busy="true">
        <div className="panel stack">
          <span className="skel" style={{ height: 20, width: "45%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      </main>
    );
  }

  if (done) {
    return (
      <main className="app-wrap narrow">
        <div className="stamp-wrap">
          <div className="stamp-box">
            <Icon name="check" size="l" />
            <h1 style={{ marginTop: 12 }}>Application submitted</h1>
            <p className="hint" style={{ marginTop: 8 }}>
              Officials will review it and you will see the decision on your candidacy page.
            </p>
            <div className="row" style={{ justifyContent: "center", marginTop: 20 }}>
              <Link className="btn primary" href="/candidate">
                Track my application
              </Link>
              <Link className="btn ghost" href="/elections">
                Back to elections
              </Link>
            </div>
          </div>
        </div>
      </main>
    );
  }

  const closed = !view.accepting_applications;
  const blocked = closed || !view.eligible;
  const selectedPosition = view.positions.find((position) => position.id === positionId);

  return (
    <main className="app-wrap">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <Link href={`/elections/${id}`}>{view.election.title}</Link>
        <span>/</span>
        <span>Become a candidate</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Candidate application</span>
          <h1>{view.election.title}</h1>
          <p>Applications close when voting opens on {formatEAT(view.election.starts_at)}.</p>
        </div>
      </div>

      {closed ? (
        <div className="banner hatch" role="alert" style={{ marginBottom: 16 }}>
          <Icon name="clock" />
          <div>
            <b>Applications are closed</b>
            {view.application_reason ??
              "Applications open when the election is in Registration and close when voting opens."}
          </div>
        </div>
      ) : null}
      {!view.eligible ? (
        <div className="banner bad" role="alert" style={{ marginBottom: 16 }}>
          <Icon name="alert" />
          <div>
            <b>You are not eligible to stand in this election</b>
            {view.eligibility_reason ?? "Only eligible voters may stand for election."}
          </div>
        </div>
      ) : null}

      {selectedPosition?.requirements ? (
        <div className="panel stack" style={{ marginBottom: 16 }}>
          <span className="lbl">Requirements for {selectedPosition.title}</span>
          <p style={{ whiteSpace: "pre-wrap" }}>{selectedPosition.requirements}</p>
        </div>
      ) : null}

      <div className="split">
        <form className="panel stack" onSubmit={submit}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Your application</h2>
            <span className="mono">all fields marked * are required</span>
          </div>

          <div className="field">
            <label htmlFor="position">Position *</label>
            <select
              id="position"
              className="sel"
              value={positionId}
              onChange={(event) => setPositionId(event.target.value)}
              disabled={blocked}
            >
              {view.positions.length === 0 ? (
                <option value="">No positions configured</option>
              ) : null}
              {view.positions.map((position) => (
                <option key={position.id} value={position.id}>
                  {position.title}
                </option>
              ))}
            </select>
            {view.positions.length === 0 ? (
              <span className="hint">
                An official has not added positions to this election yet.
              </span>
            ) : null}
          </div>

          {view.applicant ? (
            <div className="form-grid">
              <div className="field">
                <label htmlFor="programme">Programme</label>
                <input
                  id="programme"
                  className="in"
                  value={view.applicant.programme ?? ""}
                  readOnly
                />
              </div>
              <div className="field">
                <label htmlFor="study-year">Year of study</label>
                <input
                  id="study-year"
                  className="in"
                  value={view.applicant.study_year ?? ""}
                  readOnly
                />
              </div>
            </div>
          ) : null}

          <div className="form-grid">
            <div className="field">
              <label htmlFor="full-name">Name on the ballot *</label>
              <input
                id="full-name"
                className="in"
                value={fullName}
                onChange={(event) => setFullName(event.target.value)}
                disabled={blocked}
                maxLength={255}
                required
              />
            </div>
            <div className="field">
              <label htmlFor="date-of-birth">Date of birth *</label>
              <input
                id="date-of-birth"
                className="in"
                type="date"
                value={dateOfBirth}
                onChange={(event) => setDateOfBirth(event.target.value)}
                disabled={blocked}
                required
              />
            </div>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="sex">Sex *</label>
              <select
                id="sex"
                className="sel"
                value={sex}
                onChange={(event) => setSex(event.target.value)}
                disabled={blocked}
                required
              >
                <option value="">Select…</option>
                <option value="MALE">Male</option>
                <option value="FEMALE">Female</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="religion">Religion (optional)</label>
              <input
                id="religion"
                className="in"
                value={religion}
                onChange={(event) => setReligion(event.target.value)}
                disabled={blocked}
                maxLength={100}
              />
            </div>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="slogan">Slogan (optional)</label>
              <input
                id="slogan"
                className="in"
                value={slogan}
                onChange={(event) => setSlogan(event.target.value)}
                disabled={blocked}
                maxLength={200}
                placeholder="One line voters will remember"
              />
            </div>
            <div className="field">
              <label htmlFor="registration-number">Registration number</label>
              <input
                id="registration-number"
                className="in"
                value={view.applicant?.registration_number ?? ""}
                readOnly
              />
            </div>
          </div>

          <div className="field">
            <label htmlFor="bio">Biography</label>
            <textarea
              id="bio"
              className="ta"
              rows={5}
              value={bio}
              onChange={(event) => setBio(event.target.value)}
              disabled={blocked}
              maxLength={5000}
              placeholder="Who you are and why you are standing."
            />
          </div>

          <div className="field">
            <label htmlFor="manifesto">Manifesto summary</label>
            <textarea
              id="manifesto"
              className="ta"
              rows={5}
              value={manifesto}
              onChange={(event) => setManifesto(event.target.value)}
              disabled={blocked}
              maxLength={20000}
              placeholder="The promises voters will read on the ballot."
            />
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="contact-email">Contact email</label>
              <input
                id="contact-email"
                className="in"
                type="email"
                value={contactEmail}
                onChange={(event) => setContactEmail(event.target.value)}
                disabled={blocked}
                placeholder={`you@${STUDENT_EMAIL_DOMAIN}`}
              />
            </div>
            <div className="field">
              <label htmlFor="contact-phone">Phone (optional)</label>
              <input
                id="contact-phone"
                className="in"
                value={contactPhone}
                onChange={(event) => setContactPhone(event.target.value)}
                disabled={blocked}
                placeholder="+256 …"
              />
            </div>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="passport-photo">Passport photo (JPEG or PNG, max 2 MB) *</label>
              <input
                id="passport-photo"
                className="in"
                type="file"
                accept="image/jpeg,image/png"
                disabled={blocked || uploading !== ""}
                onChange={(event) => pickFile("passport", event.target.files?.[0])}
                required
              />
              {uploading === "passport" ? (
                <span className="hint">Uploading passport photo…</span>
              ) : null}
              {passportPhotoUrl ? (
                <span className="fileline">
                  <Icon name="check" size="s" />
                  Passport photo uploaded
                  <span className="mono">ready</span>
                </span>
              ) : null}
            </div>
            <div className="field">
              <label htmlFor="student-id">Student ID photo (JPEG or PNG, max 2 MB) *</label>
              <input
                id="student-id"
                className="in"
                type="file"
                accept="image/jpeg,image/png"
                disabled={blocked || uploading !== ""}
                onChange={(event) => pickFile("id", event.target.files?.[0])}
                required
              />
              {uploading === "id" ? <span className="hint">Uploading student ID…</span> : null}
              {idDocumentUrl ? (
                <span className="fileline">
                  <Icon name="check" size="s" />
                  Student ID uploaded
                  <span className="mono">ready</span>
                </span>
              ) : null}
            </div>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="transcript-file">Partial transcript (PDF, max 5 MB) *</label>
              <input
                id="transcript-file"
                className="in"
                type="file"
                accept="application/pdf"
                disabled={blocked || uploading !== ""}
                onChange={(event) => pickFile("transcript", event.target.files?.[0])}
                required
              />
              {uploading === "transcript" ? (
                <span className="hint">Uploading transcript…</span>
              ) : null}
              {transcriptUrl ? (
                <span className="fileline">
                  <Icon name="check" size="s" />
                  Transcript uploaded
                  <span className="mono">ready</span>
                </span>
              ) : null}
            </div>
            <div className="field">
              <label htmlFor="manifesto-file">Manifesto PDF (optional, max 5 MB)</label>
              <input
                id="manifesto-file"
                className="in"
                type="file"
                accept="application/pdf"
                disabled={blocked || uploading !== ""}
                onChange={(event) => pickFile("manifesto", event.target.files?.[0])}
              />
              {uploading === "manifesto" ? (
                <span className="hint">Uploading manifesto…</span>
              ) : null}
              {manifestoUrl ? (
                <span className="fileline">
                  <Icon name="check" size="s" />
                  Manifesto uploaded
                  <span className="mono">ready</span>
                </span>
              ) : null}
            </div>
          </div>

          <div className="field">
            <label htmlFor="photo">Ballot photo (optional, JPEG or PNG, max 2 MB)</label>
            <input
              id="photo"
              className="in"
              type="file"
              accept="image/jpeg,image/png"
              disabled={blocked || uploading !== ""}
              onChange={(event) => pickFile("photo", event.target.files?.[0])}
            />
            {uploading === "photo" ? <span className="hint">Uploading photo…</span> : null}
            {photoUrl ? (
              <span className="fileline">
                <Icon name="check" size="s" />
                Photo uploaded
                <span className="mono">ready</span>
              </span>
            ) : null}
          </div>

          <div className="row">
            <button className="btn primary" type="submit" disabled={busy || blocked}>
              {busy ? "Submitting…" : "Submit application"}
            </button>
            <Link className="btn ghost" href={`/elections/${id}`}>
              Cancel
            </Link>
          </div>
        </form>

        <aside className="stack">
          <div className="panel stack">
            <span className="lbl">Preview</span>
            <div className="row" style={{ alignItems: "flex-start", gap: 12 }}>
              {photoUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={photoUrl}
                  alt=""
                  width={64}
                  height={64}
                  style={{ objectFit: "cover", border: "1px solid var(--line)" }}
                />
              ) : (
                <div
                  style={{
                    width: 64,
                    height: 64,
                    display: "grid",
                    placeItems: "center",
                    border: "1px dashed var(--line-strong)",
                    color: "var(--ink-3)",
                  }}
                >
                  <Icon name="user" />
                </div>
              )}
              <div>
                <b>{fullName || "Your name"}</b>
                {slogan ? <span className="candidate-meta">{slogan}</span> : null}
                <span className="candidate-meta">
                  {view.positions.find((position) => position.id === positionId)?.title ??
                    "Position"}
                </span>
              </div>
            </div>
            {bio ? <p className="hint">{bio}</p> : null}
            {manifesto ? <p className="hint">{manifesto}</p> : null}
          </div>

          <div className="banner">
            <Icon name="info" size="s" />
            <div>
              <b>What happens next</b>
              An official reviews your documents and approves or rejects the application. You can
              withdraw it until it is reviewed.
            </div>
          </div>
        </aside>
      </div>
    </main>
  );
}

export default function CandidateApplyPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <SiteHeader />
      <Apply id={id} />
    </RequireAuth>
  );
}
