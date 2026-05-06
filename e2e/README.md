# E2E tests

This directory contains black-box end-to-end tests for the real `nsqlite` binary.

## Layout

- `harness/`: shared process bootstrap and HTTP helpers
- `tests/`: all executable E2E test suites grouped by domain
- `tests/system/`: public operational endpoints such as `/health` and `/version`
- `tests/query/`: query endpoint behavior
- `tests/auth/`: authentication and authorization behavior

## Conventions

- Keep helpers in `harness` small and focused on process bootstrapping plus HTTP ergonomics.
- Add new tests under `tests/<domain>/` instead of growing one large file.
- Each test should start its own isolated server with its own temp data dir and TCP port.
- Treat the server as a black box: talk to the public HTTP API only.
