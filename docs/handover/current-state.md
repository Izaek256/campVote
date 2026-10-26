# Current System State & Handover — campVote

## 1. Project Context
- **Project Name:** campVote (Mountains of the Moon University - MMU Campus E-Voting System)
- **Repository:** `https://github.com/Izaek256/campVote.git`
- **Current Status:** Foundation established; toolchain verified; ready for Grit Triple scaffold integration.

## 2. Architecture & Tech Stack
- **Framework:** Grit Framework v3.411.17 (`grit`)
- **Backend:** Go 1.26 + Gin + GORM
- **Frontends:** Next.js (TypeScript) with separate `apps/web` (public & voter portal) and `apps/admin` (election CMS)
- **Monorepo:** Turborepo + pnpm 12.15 workspaces
- **Database:** PostgreSQL (production / Docker Compose) + SQLite (zero-server local dev/testing)

## 3. Environment & Tooling Verified
- OS: Windows 11 AMD64
- Go: 1.26.4
- Node.js: 24.11.0
- npm: 11.6.4
- pnpm: 12.15.0
- Grit CLI: 3.411.17 (`C:\Users\Izaek\.grit\bin\grit.exe`)

## 4. Key Documentation Links
- Requirements: [`SRS.md`](../../SRS.md)
- Technology: [`STACK.md`](../../STACK.md)
- Architecture Decisions: [`ADR.md`](../../ADR.md)
- Governance: [`AGENTS.md`](../../AGENTS.md) & [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
- Requirements Tracker: [`docs/progress/requirements-tracker.md`](../progress/requirements-tracker.md)
- Cleanup Log: [`docs/maintenance/repository-cleanup.md`](../maintenance/repository-cleanup.md)
- Progress Log: [`docs/progress/development-log.md`](../progress/development-log.md)

## 5. Branching State & Workflow
- Root branch `main` is being committed with initial baseline documentation.
- `develop` branch will be established as integration base.
- All development proceeds on task branches (`feature/<issue>-<name>`) targeting `develop`.
