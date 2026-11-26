# Administrator-selected candidates

**Status:** implemented on `feature/selected-candidates-combined-eligibility`
(uncommitted at the time of writing).
**Scope:** a second candidacy path. Instead of applying through the ordinary
form, an administrator names the email addresses that may stand for a position,
and each named person accepts and publishes their own candidacy.

## Why

The ordinary path is self-service: a student applies, an officer reviews and
approves, and the candidate then appears on the ballot. Some positions are
filled by appointment, and the administrator knows the intended candidates in
advance. This path keeps the same candidate record and ballot mechanics, but
replaces "apply and wait" with "accept and publish".

## Data model

No new table. The existing `candidates` row carries the invitation:

| Field | Meaning |
|---|---|
| `invited_email` | The address the administrator named. Non-empty marks an invited candidacy. |
| `accepted_at` | When the named account accepted. |
| `approval_status` | `SELECTED` → `ACCEPTED` → `APPROVED` (auto-published). |

`positions.admin_selected` closes the position to ordinary applications and
turns on the selection panel in the staff UI.

The ordinary statuses (`NOMINATED`, `PENDING`, `APPROVED`, `REJECTED`,
`WITHDRAWN`) keep their existing meaning.

## Flow

1. **Select.** A staff member with `candidates.create` posts one or more student
   emails to `POST /api/v1/elections/:id/positions/:position_id/selections`.
   The endpoint:
   - refuses a position that is not in the election;
   - refuses after the election is closed or its start time has passed;
   - validates each address as a student registration email;
   - skips addresses that already have a non-withdrawn, non-rejected candidacy
     for the position, so re-posting a list is safe;
   - creates each invitation with status `SELECTED` and sets
     `positions.admin_selected = true`.
2. **Accept.** The signed-in account with the same verified address calls
   `POST /api/v1/candidates/my/:id/accept`. The account must have confirmed its
   email; another account gets `404`, not a hint that the invitation exists.
   Accepting sets `nominated_by`, `user_id` and `accepted_at` and moves the row
   to `ACCEPTED`.
3. **Publish.** The candidate submits their details and a manifesto file to
   `POST /api/v1/candidates/my/:id/submit`. The server:
   - requires a name, date of birth, sex and passport photo;
   - derives registration number, programme and study year from the verified
     institutional email;
   - requires a manifesto that resolves to an `Upload` owned by this account,
     with an allowed MIME type and a size up to 100 MB;
   - sets status `APPROVED`, assigns the next ballot number and writes an
     `AuditActionCandidateApproved` audit row, so publication is immediate.

The same submit endpoint is the edit path. Details and the manifesto can be
replaced until voting opens; after that the server refuses.

## Manifesto files

Accepted: `application/pdf`, `audio/mpeg`, `audio/mp4`, `video/mp4` (PDF, MP3,
MP4/M4A), up to 100 MB. The file picker checks type and size in the browser, and
the server checks the stored upload again: the picker is not the control. A
candidate cannot point the field at another account's upload or an outside URL.

Voters see the file in the page through `ManifestoViewer`: an embedded PDF
frame, an audio player or a video player. An older or unrecognised link falls
back to an "Open manifesto" link instead of rendering nothing.

## Ordinary path stays closed

For an `admin_selected` position:

- `GET /elections/:id/candidate-application` omits the position;
- `POST /elections/:id/candidates/apply` returns `403`;
- the standalone position apply route returns `403`;
- officer candidate creation for the position returns `403`.

Ordinary positions are unchanged.

## Authorization and privacy

- Inviting and listing selections requires the same staff permission as
  candidate management (`candidates.create` / `candidates.view`).
- Accepting and submitting require only an authenticated account, and the server
  matches the account's verified email to `invited_email`. No account can accept
  another person's invitation.
- The staff selection list shows the invited addresses and status; it contains no
  ballot choices.
- The audit entry records the count of selections and the candidate's name on
  publication, never a voter-to-choice mapping.

## Tests

`apps/api/internal/handlers/selected_candidate_test.go` covers:

- an administrator selects a student, and the invitation appears in
  `/candidates/my` before acceptance;
- the named account accepts, while another account gets `404`;
- an ordinary application to the selected position is forbidden;
- submission without a manifesto upload is refused (`422`);
- submission with the account's MP3 upload publishes the candidacy with a ballot
  number;
- the submit endpoint edits the details before voting opens.

The eligibility tests cover the separate combined-eligibility change:
criteria-only, listed-only, both and neither, plus the union count without
double counting and revocation behaviour. See
[`../eligibility.md`](../eligibility.md).

## Known limits

- There is no "decline" action for an invitation; the administrator can revoke
  or leave it pending.
- Unchecking `admin_selected` does not delete invitations that were already
  created.
- The selection list has no search or pagination; it is a small per-position
  list.
- Manifesto type is read from the stored upload's MIME type. The upload endpoint
  is the place that inspects the file content/extension; this feature trusts that
  record.
