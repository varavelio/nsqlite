# E2E tests

This directory contains black-box end-to-end tests for the real `nsqlite` binary.

## Layout

- `harness/`: shared process bootstrap and HTTP helpers used by every E2E layer
- `tests/`: Go E2E suites for richer scenarios that benefit from typed helpers and custom assertions
- `tests/system/`: Go tests for public operational endpoints such as `/health` and `/version`
- `tests/query/`: Go tests for query endpoint behavior
- `tests/auth/`: Go tests for authentication and authorization behavior
- `hurl/`: Hurl-based black-box scenarios plus one global Go runner that discovers `.hurl` files recursively
- `hurl/system/`: small HTTP-level checks for operational endpoints
- `hurl/query/`: small HTTP-level checks for SQL query flows
- `hurl/auth/`: small HTTP-level checks for authentication behavior

## Conventions

- Keep helpers in `harness` small and focused on process bootstrapping plus HTTP ergonomics.
- Add new Go tests under `tests/<domain>/` instead of growing one large file.
- Add new Hurl scenarios under `hurl/<domain>/` as small, black-box `.hurl` files.
- Each test should start its own isolated server with its own temp data dir and TCP port.
- Treat the server as a black box: talk to the public HTTP API only.

## Go E2E and Hurl E2E

The repository now supports two complementary E2E layers that share the same real process harness.

### Use Hurl when

- the scenario is a short request/response flow
- raw HTTP readability matters more than typed Go assertions
- you want to add a fast black-box regression test with minimal ceremony

Each `.hurl` file is executed as its own Go subtest. The global runner under `e2e/hurl/` discovers files recursively, starts one isolated NSQLite server per file, and passes common variables such as `host`, `admin_token`, `rw_token`, and `ro_token` into Hurl.

### Use Go tests when

- the scenario needs loops, branching, reusable setup, or richer assertions
- you want to decode JSON into typed structures
- the behavior spans multiple dependent calls that are clearer in Go than in Hurl

## Adding new tests

- Prefer small, single-purpose `.hurl` files for straightforward HTTP coverage.
- Keep Hurl files domain-oriented so the tree scales cleanly as the suite grows.
- Keep using `harness.StartServer` from Go tests whenever a scenario needs more logic than a single Hurl file should contain.
