# Responsive UI and Usability

**Date:** 2026-11-09  
**Branch:** `fix/ui-responsive-usability`  
**Status:** Implemented and verified; GitHub issue not yet created

## Goal

Improve the mobile layout and task flow in election management and student
candidate pages without changing election, eligibility, or ballot rules.

## Acceptance criteria

- Shared public and admin headers remain usable on narrow screens, with no
  forced page zoom or hidden primary navigation.
- Main content grids and forms reflow to the available width. Wide data tables
  scroll inside their own container instead of widening the page.
- Position ballot fields explain seats, selection limits, question types, ballot
  order, and blank selections in plain language.
- Forms revealed by an in-page action scroll into view, respecting reduced
  motion.
- Long position details and manifestos have a compact preview and a keyboard-
  accessible way to read the full text.
- Run responsive checks at 360px, 390px, 768px, 1024px, and 1440px, including
  dark mode and a landscape phone viewport.

## Implemented in this pass

- Replaced the public header's fixed 640px minimum width with a compact,
  accessible navigation menu for viewports up to 800px.
- Removed a hard-coded 560px minimum width on the election lifecycle rail. This
  was the cause of the mobile “Manage election” page overflowing to about 620px
  and forcing a zoom-out; the three stages now share the available width.
- Made single-item card grids use available space while limiting student
  position cards to a readable maximum width.
- Made page headings and admin titles tolerate long content, constrained the
  mobile navigation drawer to the viewport, and kept every data table inside a
  horizontal scroll container (the ballot-versions table was missing one).
- Clarified the election position editor's ballot question, display order,
  seats, minimum/maximum choices, and blank-selection controls, and grouped them
  under a “Ballot settings” heading so the section reads as one block.
- Scrolls the student application form, admin election creation/invitation
  forms, and election position editor into view when opened.
- Added a direct link to the candidate-manifesto section and expandable views
  for long position details and manifestos.
- Added tests for the compact header interaction and reduced-motion-aware
  scrolling.

## Second pass: long text and uploads

A follow-up report described three more places where long text distorted the
interface. Root causes and fixes:

- The candidate review card rendered position requirements at full length, so
  one long record grew the card and pushed the entries below it down. Slogans,
  reviewer notes, and requirements are now clamped, with the full requirements
  available behind an expander so reviewers can still read them.
- The admin positions table printed the full description and requirements in a
  cell, stretching rows. Both are now clamped to two lines with a “Full
  requirements” expander, and table meta text is capped at 40ch.
- The candidate application form overflowed by about 700px as soon as a file was
  uploaded: the filename used `white-space: nowrap`, giving it a ~980px
  min-content width. That min-content propagated through `1fr` grid tracks
  (`.form-grid`, `.split`) whose automatic minimum cannot shrink. The filename
  is now a single-line clamp that can break anywhere, the file-picker row wraps
  its actions onto a second line when space is tight, and those grid tracks use
  `minmax(0, 1fr)` so no long content can inflate them again.
- User-supplied field values now wrap anywhere, so one long unbroken token can
  no longer widen a card or table.

## Scope and follow-up

The changes address the reported headers, election/position management,
candidate application, and manifesto flows, plus shared responsive primitives.
A headless-Chromium sweep of the public, student, and admin routes found no
remaining horizontal overflow at phone and desktop widths; see Verification for
the exact coverage. The sweep used synthetic API fixtures, not a seeded
production-like election, so a final check against real demo data is still
sensible before release.

The required GitHub issue has not been created because GitHub CLI (`gh`) is not
available in this environment. Create an issue titled **“Improve responsive
layouts and clarify election task flows”**, using the goal and acceptance
criteria above, then link the branch/PR to it.

## Verification

Run on 2026-11-09 from the repository root.

| Check | Command | Result |
|---|---|---|
| Format and lint | `pnpm format:check` | Passed — 238 files, no fixes |
| Frontend types | `pnpm --filter @campvote/web type-check` | Passed |
| Frontend unit tests | `pnpm --filter @campvote/web exec vitest run --maxWorkers=1 --no-file-parallelism` | Passed — 8 files, 44 tests |
| Production build | `pnpm --filter @campvote/web build` | Passed — 19 pages generated |

Responsive overflow sweep (headless Chromium against a local production build,
API responses mocked with synthetic fixtures, measuring
`documentElement.scrollWidth` against `clientWidth`):

- Public and student routes (`/`, `/how-voting-works`, auth pages,
  `/elections`, `/positions`, `/candidate`, `/results`, `/account`, privacy
  page) at 360px and 768px — 26 checks, no overflow.
- Admin routes (`/staff`, `/staff/landing`, election details, positions,
  candidates, monitor, results, results history, turnout, ballot, ballot
  versions, ballot preview) at 360, 390, 768, 1024 and 1440px — 60 checks, no
  overflow.
- Public mobile header on an election page at 360px: menu toggles
  `aria-expanded`, the account link is reachable, Escape closes it and restores
  the collapsed state, and a navigation link routes away.
- Mobile admin drawer at 360px: opens within the viewport (207px wide) and
  closes on Escape.

Long-text and upload verification (headless Chromium, production build,
synthetic fixtures containing deliberately hostile content — 2,000-character
requirements, 300-character slogans, 180-character unbroken registration
numbers, and long file names):

- Candidate review and positions pages at 360, 390, 768, 1024, and 1440px — no
  overflow. The review card with extreme content rendered at 523px tall, with
  requirements clamped to three lines and an expander present. A positions row
  holding ~3,000 characters rendered at 183px tall with both meta lines clamped.
- Candidate application form at 360, 390, 768, and 1440px, before upload and
  after uploading five documents with very long file names — no overflow
  (previously ~705px of overflow at 360px). The filled picker wraps to two lines
  at 360px rather than widening the page.

Limitations: the unit tests were run with a single worker because the default
parallel Vitest run exceeded the environment's worker-startup budget. Dark-mode
appearance was not captured as screenshots.

