# Kickoff checklist for the first development session

1. Clone `https://github.com/Izaek256/campVote.git` and inspect its branches and existing files before changing anything.
2. Confirm `main` is stable and `develop` is the integration branch; protect both and require PR checks/reviews where supported.
3. Copy/merge the files in this package into the repository root and `docs/` without overwriting the generated Grit app configuration.
4. Read `AGENTS.md`, `CONTRIBUTING.md`, `SRS.md`, `STACK.md`, and `ADR.md`.
5. Verify Grit CLI version and supported Triple/Next.js options using its installed help and official docs. Generate the scaffold in a temporary directory first.
6. Open GitHub issues for Phase 0 and each unresolved item in `docs/implementation/OPEN_QUESTIONS.md`.
7. Create a feature branch from `develop` for the first issue; do not commit directly to `main` or `develop`.
8. Record actual setup/version details and test commands in `docs/implementation/SETUP_LOG.md`.
9. Add CI that runs real checks only after the scaffold structure is known.
10. Do not use real student data or treat the prototype as certified for actual elections.
