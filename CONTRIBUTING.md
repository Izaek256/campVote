# Contributing to campVote

These guidelines apply to humans and AI coding agents. `AGENTS.md` contains the detailed development rules. Repository-owner decisions and GitHub branch protections take precedence.

## 1. Before starting

1. Read `AGENTS.md`, `README.md`, and relevant documents in `docs/`.
2. Search GitHub Issues for an existing issue.
3. For substantial work, open an issue with context, expected behavior, acceptance criteria, and sanitized evidence.
4. Run `git status --short --branch` and preserve existing uncommitted work.
5. Inspect current implementation and tests before changing code.

## 2. Branching strategy

`main` is stable/release. `develop` is staging/integration. All normal work uses a task branch and pull request.

| Type | Branch pattern | Start from | PR target |
|---|---|---|---|
| Feature | `feature/<issue>-<short-name>` | `develop` | `develop` |
| Bug fix | `fix/<issue>-<short-name>` | `develop` | `develop` |
| Documentation | `docs/<issue>-<short-name>` | `develop` | `develop` |
| Tests/CI | `test/<issue>-<short-name>` or `ci/<issue>-<short-name>` | `develop` | `develop` |
| Emergency production fix | `hotfix/<issue>-<short-name>` | `main` | `main` |

Examples: `feature/24-election-creation`, `fix/31-duplicate-ballots`, `docs/18-local-setup`, `hotfix/55-login-outage`.

### Required workflow

1. Fetch remote branches and update the latest `develop` for ordinary tasks.
2. Create a dedicated branch for one issue or cohesive task.
3. Implement, test, and document the change.
4. Commit on the task branch using the convention below.
5. Push the branch and open a PR into `develop`.
6. Address review comments and wait for required CI checks.
7. A maintainer reviews and merges; do not merge your own PR unless authorized.
8. Validate the integrated result on `develop`.
9. For a release, open a separate PR from `develop` into `main`.
10. For a hotfix, branch from `main`, merge via PR, then promptly open a follow-up PR to sync it into `develop`.

Never push directly to `main` or `develop`. Never force-push shared branches or bypass required reviews/CI.

Example commands (use actual checks documented in the repository):

```bash
git fetch origin
git switch develop
git pull --ff-only origin develop
git switch -c feature/24-election-creation

# Make changes, then run relevant documented checks.
git status
git add path/to/changed-files
git commit -m "feat(elections): add election creation"
git push -u origin feature/24-election-creation
```

## 3. Commit message convention

Use Conventional Commits:

```text
<type>(<scope>): <imperative summary>
```

Keep the summary specific and preferably under 72 characters. Use the body for context and a footer for issue references or breaking changes.

Types:
- `feat`: new feature
- `fix`: bug fix
- `docs`: documentation
- `test`: tests
- `refactor`: restructuring without intended behavior change
- `perf`: performance improvement
- `style`: formatting only
- `build`: build/dependency changes
- `ci`: CI workflow
- `chore`: maintenance
- `revert`: revert a commit

Examples:

```text
feat(elections): add election creation workflow
fix(voting): reject duplicate ballot submissions
docs(setup): explain Docker and manual installation
test(voting): cover concurrent duplicate submissions
ci(github): run backend and frontend checks
```

Use imperative verbs (`add`, `fix`, `document`). Avoid vague messages such as `updates`, `stuff`, `final`, or `AI changes`. Keep unrelated changes in separate commits. Never claim tests passed if they did not run. Reference issues where useful, e.g. `Closes #24` in the PR body. Do not rewrite published shared history without maintainer approval.

## 4. Pull request requirements

Every PR needs a clear title and useful description. Use `.github/PULL_REQUEST_TEMPLATE.md` if available. Include:

- **Summary:** what changed and why.
- **Issue:** link it; use `Closes #<number>` when fully resolved.
- **Scope:** what is and is not included.
- **Implementation:** design choices, API/UI changes, migrations, and dependencies.
- **Testing:** exact commands and actual results (passed, failed, or blocked).
- **Screenshots:** for meaningful UI changes when useful.
- **Configuration/setup:** changed environment variables, seed data, or manual steps.
- **Documentation:** links to updated docs.
- **Risks/follow-up:** known limitations and deferred work.

A PR is not ready if relevant tests or docs are missing, generated files are unexplained, or unrelated changes are included.

### Review and merge policy

- PRs into `develop` require review and passing required CI.
- Normal PRs into `main` come from `develop`; approved emergency hotfixes are the exception.
- Resolve or discuss review comments; do not silently dismiss security/correctness feedback.
- Do not merge with failing required checks. Report external/environmental blockers and wait for maintainer direction.
- Configure GitHub branch protection to block direct pushes and merges that bypass checks; documentation alone cannot enforce this.

## 5. Tests and CI/CD

Before opening a PR, run the relevant commands documented in `README.md`. Depending on the change, checks may include formatting, linting, Go tests, frontend lint/type checks/tests, PostgreSQL integration tests, migrations on a clean database, production builds, and Docker Compose checks.

CI should run essential checks for PRs into `develop` and `main`. A check passes only when the workflow reports success. Never remove tests, weaken assertions, or mark tests skipped just to get a green build.

Critical voting tests should cover eligibility, election state, duplicate/concurrent submissions, invalid choices, tally correctness, authorization, and privacy of voter-choice mappings.

## 6. Issues and bug reports

Use GitHub Issues for bugs, features, technical debt, blockers, and deferred work. Search first to avoid duplicates.

Include a clear title, context, expected/actual behavior, reproduction steps, environment, sanitized logs/screenshots, acceptance criteria, and impact where known. Never include passwords, tokens, real student/staff data, individual ballot choices, or production database dumps. Report suspected vulnerabilities privately using the maintainer's preferred security contact or GitHub private vulnerability reporting if enabled.

AI agents should file/update issues for meaningful unrelated defects rather than expanding a PR silently. If an agent cannot access GitHub Issues, it must provide a ready-to-copy issue draft and state that it did not create the issue.

## 7. Documentation

Documentation is part of the contribution:

- Update root `README.md` when setup, commands, configuration, ports, or project status changes.
- Keep feature docs in `docs/features/`.
- Record architecture/dependency decisions in `docs/ADR.md` or `docs/decisions/`.
- Record investigations and verified fixes in `docs/debugging/`.
- Document testing/CI changes in `docs/testing/` where useful.
- Document environment, migration, Docker, and deployment changes in `docs/operations/`.
- Update `docs/README.md` when docs are added or renamed.
- For substantial work, update progress/handover notes with status, tests, known issues, and next steps.

Keep docs accurate and avoid duplicating the same instructions in several places.

## 8. Environment variables and secrets

- Add required variables to `.env.example` with safe sample values and explanations.
- Never commit populated `.env` files, credentials, tokens, private keys, or production connection strings.
- Never print secrets into logs, CI artifacts, screenshots, issues, or docs.
- Mark demo credentials/data as local-only.
- Update setup docs when configuration changes.

## 9. Code, security, and voting privacy

- Follow `AGENTS.md`, architecture decisions, language conventions, and existing formatters.
- Keep changes focused, readable, and testable.
- Validate sensitive operations on the server; frontend checks are not security controls.
- Use ORM-safe database access and explicit authorization.
- Never disclose who voted for which candidate; separate participation records from ballot selections.
- Use synthetic data only. Email-domain checks alone do not prove faculty, department, class, or year eligibility.
- Do not claim this academic project is certified for official high-stakes elections.
- Document known risks and deferred security features honestly.

## 10. Ready-to-merge checklist

- [ ] Correct branch/base and issue.
- [ ] No unrelated changes.
- [ ] Formatting, linting, type checks, tests, and builds run as applicable.
- [ ] Tests added/updated for behavior changes.
- [ ] Migrations, authorization, and voting privacy reviewed where relevant.
- [ ] README, docs, and `.env.example` updated where needed.
- [ ] No secrets, personal data, local DB files, or build artifacts included.
- [ ] Conventional commit messages used.
- [ ] PR summary, issue link, test evidence, and risks provided.
- [ ] Required CI checks pass and review feedback is resolved.
