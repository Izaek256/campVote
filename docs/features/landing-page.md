# Landing Page (ISSUE-31)

**Branch:** `feature/31-landing-page`
**Status:** Implemented (backend + frontend), tests passing
**Related docs:** [`../ui-views.md`](../ui-views.md), [`../../.env.example`](../../.env.example)

---

## Summary

A public landing page at `/` that presents the campus election platform to
visitors: hero headline, announcement banner, scrolling ticker, "How it
works" steps, feature grid, FAQ, and footer. An administrator edits the copy
through a staff-only CMS at `/staff/landing` and publishes it when ready. The
public page reads a published snapshot, never the live draft, so editing and
previewing never change what visitors see until an explicit publish.

Signed-in users are redirected away from the landing page to their dashboard,
so the page is a visitor-first surface.

---

## Architecture

```
Admin edits draft  →  saves (no public change)  →  publishes (snapshot)
                                                        ↓
                                          LandingSettings.PublishedJSON
                                                        ↓
                              GET /api/v1/landing        ← public reads this
                              GET /api/v1/landing/upcoming
                                                        ↓
                                          Next.js page.tsx (ISR, 60 s)
```

The landing page uses Next.js incremental static regeneration with a 60-second
revalidate window, so content changes go live within a minute of an
administrator publishing.

---

## Backend

### Package `internal/landing`

- **`landing.go`** — `Document` struct (the full editable content), `Default()`
  (the built-in content a fresh install ships with), `Normalize()` (trims
  whitespace), `IconKeys` (allowlist of eight feature illustrations).
- **`validate.go`** — `Validate(d) map[string]string`: per-field validation
  returning a map keyed by dotted field path. Enforces plain text only (no
  angle brackets, no control characters beyond `\n`), bounded lengths, links
  limited to site-relative paths or `https://`, banner time-window ordering,
  and icon allowlist membership.

### Models (`internal/models/landing.go`)

Three GORM tables, migrated via `AutoMigrate`:

| Table | Key fields | Notes |
|---|---|---|
| `LandingSettings` | Hero, banner, ticker, how-steps, footer, `PublishedJSON`, `PublishedAt` | Single row, fixed id `"landing"` (`LandingSettingsID`). `PublishedJSON` holds the last published snapshot; the public endpoint reads it, not the draft columns. |
| `LandingFeature` | `IconKey`, `Title`, `Body`, `SortOrder`, `Visible` | Draft feature cards. `Visible` controls per-card draft visibility. |
| `LandingFAQ` | `Question`, `Answer`, `SortOrder`, `Visible` | Draft FAQ items. |

The `Election` model gains a `public_listing bool` field; elections with
`public_listing = true` and `status = 'DRAFT'` (scheduled) appear in the
upcoming-elections strip.

### Migration (`internal/migrations/add_landing_indexes.go`)

Adds a partial index for the upcoming-elections query:

```sql
CREATE INDEX IF NOT EXISTS idx_elections_public_upcoming
ON elections (starts_at)
WHERE public_listing AND status = 'DRAFT'
```

The predicate keeps the index small — only elections explicitly published to
the landing page are included.

### Handlers (`internal/handlers/landing.go`)

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/api/v1/landing` | GET | public | Returns the published `Document` (or built-in defaults if nothing published yet). |
| `/api/v1/landing/upcoming` | GET | public | Returns elections with `public_listing = true` and `status = 'DRAFT'`, projected to a safe summary (id, title, starts_at, ends_at). |
| `/api/v1/landing/admin` | GET | admin | Returns the draft document with per-feature and per-FAQ visibility flags. |
| `/api/v1/landing/admin` | PUT | admin | Saves the draft (validates first; rejects with 422 + field map on error). |
| `/api/v1/landing/admin/publish` | POST | admin | Snapshots the current draft into `PublishedJSON` and stamps `PublishedAt` / `PublishedByID`. |
| `/api/v1/landing/admin/revert` | POST | admin | Discards the draft and reloads it from `PublishedJSON` (or defaults). |

Save, publish, and revert each run inside a transaction so partial writes
cannot leave the draft in an inconsistent state.

### Seeder (`internal/database/landing_seeder.go`)

Seeds the landing settings with built-in defaults on first run (when no
`LandingSettings` row exists). Subsequent runs leave existing content
untouched, so admin edits survive reseeding.

### Tests

- `internal/handlers/landing_test.go` — 9 tests covering: public read with and
  without published content, upcoming elections projection, admin get/save
  with validation, publish, revert, and authorization (non-admin rejected).
- `internal/database/landing_seeder_test.go` — verifies seeder idempotency.
- `internal/migrations/add_landing_indexes_test.go` — verifies the index is
  created and is queryable.

---

## Frontend

### Public page (`app/page.tsx`)

Server component with `revalidate = 60` (ISR). Fetches the landing document and
upcoming elections in parallel. Redirects signed-in users to their dashboard
via `LandingSignedInRedirect`. Generates `Metadata` for SEO and social previews
using `SITE_URL` for canonical/OG URLs and `/og.png` for the preview image.

### Admin CMS (`app/staff/landing/`)

Staff-only editor with:
- Hero, banner (with time window), ticker, how-it-works steps, feature grid,
  FAQ, and footer editing.
- Per-feature and per-FAQ visibility toggles.
- Save (draft), Publish, and Revert actions.
- Inline validation errors keyed to fields.
- Unsaved-changes guard.

### Components (`components/landing/`)

- `landing-page.tsx` — renders the public page from a `Document`.
- `landing-redirect.tsx` — client guard redirecting signed-in users.
- Admin editor components for each section.

### Library

- `lib/landing.ts` — API client functions (`fetchLanding`, `fetchUpcoming`,
  admin save/publish/revert).
- `lib/config.ts` — `SITE_URL` (from `NEXT_PUBLIC_SITE_URL`, falling back to
  Vercel URL or localhost), `CAMPUS_NAME`, `ELECTION_AUTHORITY`.
- `lib/types.ts` — shared `LandingDocument` type.
- `lib/api.ts` — landing endpoint bindings.

### Styling

- `app/landing.css` — landing-page-specific styles.
- `public/og.png` — 1200×630 social-preview image.

---

## Environment variables

| Variable | Scope | Purpose |
|---|---|---|
| `NEXT_PUBLIC_SITE_URL` | frontend | Absolute origin for canonical and OG URLs. Defaults to `http://localhost:3000`; set in deployment. |
| `NEXT_PUBLIC_CAMPUS_NAME` | frontend | Campus name shown in the hero and page title. |
| `NEXT_PUBLIC_ELECTION_AUTHORITY` | frontend | Authority name in the footer. |
| `NEXT_PUBLIC_STUDENT_EMAIL_DOMAIN` | frontend | Student email domain shown in signup prompts. |
| `NEXT_PUBLIC_STAFF_EMAIL_DOMAIN` | frontend | Staff email domain shown in staff sign-in. |

All are documented in `.env.example`.

---

## Security and privacy

- **Plain text only**: the validator rejects angle brackets and control
  characters; the renderer escapes as well (defence in depth). An administrator
  cannot inject markup into the landing page.
- **Link safety**: banner links must be a site-relative path (`/...`) or an
  absolute `https://` URL. `javascript:`, `data:`, and `http:` are blocked.
- **Icon allowlist**: feature icons must be one of eight predefined keys; the
  CMS cannot point at a non-existent illustration.
- **No fabricated figures**: the landing page never prints voter counts,
  turnout percentages, or ballot numbers it cannot source from live election
  data. The default content omits the prototype's sample statistics.
- **Draft/publish separation**: the public endpoint reads a published
  snapshot, not the live draft, so an admin can edit and preview without
  accidentally publishing incomplete or wrong content.
- **Authorization**: all admin endpoints require the ADMIN role; non-admins
  receive 403.
- **Upcoming elections**: only elections the admin explicitly marked
  `public_listing = true` appear; the projection strips everything except
  id, title, and time bounds. No voter data is exposed.

---

## Verification

| Check | Command | Result |
|---|---|---|
| Go vet | `go vet ./...` | pass |
| Go tests (landing) | `go test ./internal/handlers/ -run Landing` | 9/9 pass |
| Go tests (seeder) | `go test ./internal/database/ -run Landing` | pass |
| Go tests (migration) | `go test ./internal/migrations/ -run Landing` | pass |
| TypeScript | `npx tsc --noEmit` (apps/web) | pass |
| Biome lint | `npx biome lint` (landing source files) | pass |
| Next.js build | `next build` (apps/web) | pass |

Pre-existing `internal/services` CGO/sqlite failures are unrelated to this
feature (environment-only issue with `CGO_ENABLED=0`).
