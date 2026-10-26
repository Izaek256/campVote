# CI workflow setup note

Create runnable CI workflows after the pinned Grit scaffold is generated and its real workspace paths and commands are known. The workflow must run format/lint/type checks, unit tests, migrations and database integration tests, frontend/build checks, and dependency/security checks as applicable.

Do not add a placeholder workflow that always passes. Record the exact checks and required branch protection settings in the repository documentation, and verify that PRs into `develop` and `main` cannot merge with failing required checks.
