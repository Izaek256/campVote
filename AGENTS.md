# AI Agent Development Rules — campVote

These rules apply to every AI coding agent working on this repository. Read them before planning, implementing, debugging, reviewing, or refactoring.

## 1. Non-negotiable rules

1. Never work directly on `main` or `develop`. Use task branches and pull requests.
2. Inspect the repository, docs, tests, configuration, and Git status before editing.
3. Preserve uncommitted work. Never run destructive Git commands, force-push, or rewrite shared history without explicit maintainer approval.
4. Add or update tests for behavior changes. Never claim a check passed unless it actually ran and passed.
5. Document features, decisions, debugging, tests, and handovers in `docs/` as work progresses.
6. Never commit secrets, populated `.env` files, tokens, private keys, production data, or real student/staff personal information.
7. Report status accurately: planned, in progress, implemented, tested, blocked, or deferred.
8. Keep solutions appropriate for an academic assignment. Prefer understandable code over unnecessary infrastructure.
9. Protect ballot privacy: never expose a voter's identity alongside their ballot choices in public APIs, logs, analytics, realtime events, or candidate-facing views.
10. Do not silently broaden scope or make consequential architecture/security decisions. Document options and ask the maintainer when needed.

## 2. Required steps before each task

1. Read `AGENTS.md`, `CONTRIBUTING.md`, `README.md`, and relevant files in `docs/`.
2. Run `git status --short --branch`; confirm the current branch and preserve existing work.
3. Fetch/inspect remote branches when available; do not assume local branches are current.
4. Find the relevant GitHub issue. For substantial tasks, create or request an issue with acceptance criteria.
5. Inspect relevant implementation, tests, configuration, and architecture decisions.
6. Write a short plan identifying intended changes, files likely to change, risks, and checks.
7. Follow existing Grit-generated conventions. Do not replace the structure blindly.

## 3. Branching and pull requests

```text
main (stable/release)
└── develop (staging/integration)
    ├── feature/<issue>-<short-name>
    ├── fix/<issue>-<short-name>
    ├── docs/<issue>-<short-name>
    └── test/<issue>-<short-name>

Emergency production fix:
main
└── hotfix/<issue>-<short-name>
    └── PR → main, then follow-up PR to sync the fix into develop
```

- Normal task branches start from the latest `develop`.
- Feature/fix/docs/test branches require a PR into `develop`.
- A release uses a separate PR from `develop` into `main`.
- A production hotfix branches from `main`, targets `main`, and must be propagated back into `develop`.
- Never push directly to `main` or `develop`.
- Never merge your own PR unless the repository owner explicitly authorizes it.
- Never force-push shared branches or bypass required review/CI checks.
- **All branches must pass CI/CD tests before merging.** The CI workflow (`.github/workflows/ci.yml`) runs backend tests, frontend type checks, linting, and builds. A PR cannot merge into `develop` or `main` unless all required checks pass.
- Follow `CONTRIBUTING.md` for exact workflow and commit conventions.

## 4. Clear, understandable file structure

Keep the repository approachable for new students and contributors. If a structure already exists, understand and preserve it; propose large changes before moving files.

Suggested target (adapt to the actual Grit template):

```text
campVote/
├── AGENTS.md
├── CONTRIBUTING.md
├── README.md
├── .env.example
├── .gitignore
├── compose.yaml
├── docs/
│   ├── README.md
│   ├── SRS.md
│   ├── ADR.md
│   ├── STACK.md
│   ├── architecture/
│   ├── features/
│   ├── debugging/
│   ├── testing/
│   ├── operations/
│   ├── decisions/
│   ├── progress/
│   └── handover/
├── backend/
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   ├── tests/
│   └── go.mod
├── frontend/
│   ├── app/ or src/
│   ├── components/
│   ├── lib/
│   ├── public/
│   ├── tests/
│   └── package.json
├── scripts/
└── .github/
    ├── PULL_REQUEST_TEMPLATE.md
    ├── ISSUE_TEMPLATE/
    └── workflows/ci.yml
```

The actual Grit-generated structure may differ. This is guidance, not an instruction to move files without reason. Keep frontend, backend, migrations, tests, docs, and scripts easy to find. Use idiomatic Go packages and clear TypeScript boundaries. Avoid giant files, unnecessary folders, duplicate docs, and unexplained file moves. Keep schema migrations versioned and reproducible.

## 5. Documentation is part of implementation

Update relevant documentation in the same branch and PR as the code. Root `README.md` is the main quick-start guide; `docs/` is the detailed technical record.

| Change | Documentation expected |
|---|---|
| Feature or behavior | `docs/features/<feature>.md` or an existing feature doc |
| Architecture/dependency decision | `docs/ADR.md` or `docs/decisions/ADR-<number>-<slug>.md` |
| Bug investigation/fix | `docs/debugging/<issue>-<slug>.md` |
| Test strategy or CI change | `docs/testing/` or relevant workflow notes |
| Environment/setup change | `.env.example`, `README.md`, relevant operations docs |
| Schema/migration | Migration and relevant data-model docs |
| Docker/deployment | `README.md` and `docs/operations/` |
| Significant milestone or blocked task | `docs/progress/` and/or `docs/handover/` |
| API change | API docs/examples and tests |
| Privacy/security change | Relevant design docs and tests |

For substantial tasks, record date, issue/PR, goal and acceptance criteria, branch/status, files changed, decisions, exact checks and results, known bugs/risks, deferred work, blockers, and next step. Keep `docs/README.md` as an index and update it when documents are added or renamed. Avoid logs for trivial edits. Never describe planned work as implemented.

## 6. README and local setup

The root README must accurately explain the actual implementation and include:

- Purpose, academic scope, current status, features, stack, and high-level architecture.
- Prerequisites and supported versions.
- Docker/Compose setup: environment configuration, startup, migrations, seed data, health checks, logs, shutdown, and a clearly warned local-data reset.
- Manual setup: PostgreSQL configuration, separate frontend/backend startup, migrations, demo seeds, and tests.
- Required environment variables and link to `.env.example`.
- Exact format, lint, type-check, test, build, and migration commands.
- Local URLs/ports, demo-data instructions, troubleshooting, and links to `CONTRIBUTING.md` and `docs/`.
- A warning that demo accounts/data are for local development only.

Do not invent commands or claim setup works until verified against the actual repository. Never include real credentials in examples.

## 7. Environment and secrets

- Maintain `.env.example` with every required variable, safe placeholders, and explanations.
- Ignore populated `.env` files, credentials, tokens, private keys, sensitive local databases, logs, dependency folders, and build output as appropriate.
- Validate required config at startup; errors may identify missing variable names but must not print secret values.
- Keep development, test, and production configuration separate. Never default to production credentials.
- Use synthetic demo records only.
- Document variable purpose, required/optional status, and setup.
- Add infrastructure only when justified by project needs.

## 8. Code quality

- Follow idiomatic Go and TypeScript conventions and configured formatters/linters.
- Use clear names, small functions, explicit error handling, and understandable abstractions.
- Avoid duplicate business rules, magic values, unnecessary dependencies, and unrelated changes.
- Validate untrusted input on the server even if the frontend validates it.
- Use ORM-safe/parameterized database access; never concatenate untrusted input into SQL.
- Enforce authorization server-side. Return useful errors without exposing stack traces or internals.
- Make UI accessible and responsive, with labels, keyboard support, loading/error/empty states, and confirmation for destructive actions.
- Never silence compiler, lint, test, or type errors just to make CI green.
- Review generated code for authorization, validation, transactions, and error handling.
- Do not leave fake success responses or dead controls. Clearly label and document deferred functionality.

## 9. Campus voting integrity and privacy

This is an academic campus-election application; do not claim it is certified for official high-stakes elections.

- Enforce eligibility on the server. Domain checks (`@std.must.ac.ug` for students and `@must.ac.ug` for staff/official accounts) do not prove faculty, department, class, or year membership. Use trusted eligibility attributes or clearly marked demo records.
- Enforce one accepted ballot per eligible voter per election with database constraints and transactional server-side logic. A disabled button alone is not enough.
- Keep voter participation/eligibility records separate from ballot choices. Public, candidate-facing, and admin views must not expose a voter-to-choice mapping.
- Never log ballot choices alongside email, student/staff number, account ID, IP address, or other direct identifiers.
- Apply least-privilege server-side authorization for administrators, election officers, voters, candidates, and public viewers as appropriate.
- Validate election status/time, ballot structure, candidate eligibility, and choice rules before accepting a ballot.
- Use transactions and integrity constraints to prevent partial writes and duplicate-vote races.
- Make results visibility follow the election's configured policy; do not assume results must always be public before closing.
- Audit administrative actions without recording individual ballot choices.
- Use synthetic demo data only. Document limitations honestly. Cryptographic end-to-end verifiability, coercion resistance, independent certification, and formal election audits are out of scope unless approved as new requirements.

## 10. Testing and CI/CD

A feature is not complete until relevant tests are added/updated and checks are run. CI must run essential checks for PRs into `develop` and `main`, and on protected branches.

- Test business rules, validation, permissions, migrations, and API/database integration.
- Cover valid voting, ineligible voters, duplicate ballots, concurrent duplicate submissions, closed/not-yet-open elections, invalid choices, tally correctness, and ballot privacy.
- Test frontend success, loading, empty, and error states; add end-to-end tests for critical flows when practical.
- Run formatters, linters, type checks, tests, integration checks, and production builds as applicable.
- Prefer reproducible PostgreSQL test services and clean databases; tests must not depend on a developer's local DB state.
- Do not skip tests or weaken assertions just to pass CI. Report environmental blockers honestly.
- Do not merge with failing required checks without explicit maintainer approval of a documented exception.

## 11. Debugging protocol

1. Reproduce the issue or explain why it cannot be reproduced.
2. Record expected vs actual behavior, steps, environment, and sanitized errors.
3. Inspect logs, tests, migrations, configuration, and relevant changes before editing.
4. Form a testable root-cause hypothesis.
5. Add a regression test that fails before the fix when practical.
6. Make the smallest correct fix; do not swallow errors or disable validation to hide symptoms.
7. Run the regression test and relevant broader checks.
8. Review side effects, authorization, and privacy.
9. Document reproduction, root cause, fix, verification, and remaining limitations under `docs/debugging/`.
10. Link the issue/PR and state the next step if still blocked.

Do not say “fixed” if only the symptom was hidden or the relevant test was not run.

## 12. Code review

Review the actual diff for requirements, correctness, security, privacy, authorization, migration safety, readability, accessibility, tests, documentation, and scope. Prioritize findings as critical/high/medium/low. For each finding, state location, impact, reasoning/reproduction, and a practical fix. Separate confirmed defects from potential risks and stylistic preferences. Do not approve based only on a successful build. Do not silently fix unrelated issues during review. If no issues are found, state what was reviewed and which checks were not run.

## 13. Completion and handover checklist

- [ ] Correct task branch/base; no unrelated changes.
- [ ] Acceptance criteria met or remaining gaps documented.
- [ ] Tests cover changed behavior and important failure cases.
- [ ] Formatter/linter/type checks/tests/build run where applicable.
- [ ] Authorization, privacy, and migration effects reviewed.
- [ ] Docs and `.env.example` updated where needed.
- [ ] No secrets, real personal data, local DB files, or build artifacts committed.
- [ ] Meaningful commit messages and complete PR description.
- [ ] Issue and PR linked.
- [ ] Handover lists changes, exact test results, risks, blockers, and next step.

When blocked, stop before risky assumptions. Report evidence, attempted steps, and the smallest decision needed from the maintainer.
