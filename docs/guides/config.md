# Container Configuration

All configuration is performed through **environment variables**.

## Server Configuration

| Environment Variable          | Description                                                       | Default   |
| ----------------------------- | ----------------------------------------------------------------- | --------- |
| `NSQLITE_DATA_DIR`            | Directory where NSQLite stores its SQLite database files.         | `/data`   |
| `NSQLITE_LISTEN_HOST`         | Host address for the HTTP server to bind to.                      | `0.0.0.0` |
| `NSQLITE_LISTEN_PORT`         | TCP port for the HTTP server. Valid range: `1`–`65535`.           | `9876`    |
| `NSQLITE_MAX_REQUEST_SIZE_MB` | Maximum HTTP request body size (in MB) for the `/query` endpoint. | `100`     |

> **Validation:** `NSQLITE_LISTEN_HOST` must be a valid host address. `NSQLITE_LISTEN_PORT` must be within `1`–`65535`.

## Authentication

NSQLite supports three tiers of access control. Each variable accepts **one or more tokens** separated by whitespace. Tokens can be:

- **Plaintext** (e.g., `my-secret-token`)
- **bcrypt hashes** (e.g., `$2a$...`)
- **Argon2id hashes** (e.g., `$argon2id$...`)

| Environment Variable    | Role           | Description                                                                                                                                        |
| ----------------------- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `NSQLITE_AUTH_TOKEN`    | **Admin**      | Full access to all endpoints, including `/stats`, `/version`, and all query types (read, write, transactions).                                     |
| `NSQLITE_AUTH_TOKEN_RW` | **Read/Write** | Access to `/query` only. Can execute read queries, write queries, and transactions. Cannot access `/stats` or `/version`.                          |
| `NSQLITE_AUTH_TOKEN_RO` | **Read-Only**  | Access to `/query` only. Can only execute read queries (SELECT, etc.). Cannot execute write queries (INSERT, UPDATE, DELETE, DDL) or transactions. |

> **When all three are empty/unset:** Authentication is **disabled** — every request is treated as Admin. **Do not use in production without auth.**

> **Performance note:** Tokens are resolved on first use and cached in memory via a SHA-256 derived key, so repeated authentication (including bcrypt/Argon2 verification) happens only once per unique token.

## SQLite Tuning

| Environment Variable      | CLI Flag            | Description                                                                                                                                                                                                             | Default |
| ------------------------- | ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `NSQLITE_TX_IDLE_TIMEOUT` | `--tx-idle-timeout` | Maximum idle duration for an open transaction. If a transaction has no activity within this window, it is automatically rolled back. Valid time units: `ns`, `us`/`µs`, `ms`, `s`, `m`, `h`. Must be greater than zero. | `10s`   |
| `NSQLITE_MAX_READ_CONNS`  | `--max-read-conns`  | Maximum number of concurrent read-only SQLite connections. Higher values improve parallel read throughput but consume more memory.                                                                                      | `10`    |
| `NSQLITE_CACHE_SIZE_KB`   | `--cache-size-kb`   | SQLite page cache size **per connection**, in kilobytes. Specify the positive KB value (it is converted internally to SQLite's negative page-count representation).                                                     | `20000` |
| `NSQLITE_BUSY_TIMEOUT`    | `--busy-timeout`    | Amount of time SQLite waits for the database lock when another writer is active. Valid time units: `ns`, `us`/`µs`, `ms`, `s`, `m`, `h`. Must be greater than zero.                                                     | `5s`    |

## Litestream (Container Only)

These variables are **only evaluated by the Docker entrypoint** (`cmd/entrypoint/main.go`). They configure automatic continuous backup of the SQLite database to an S3-compatible object store via [Litestream](https://litestream.io/).

Litestream is an optional runtime wrapper. When disabled, the container runs NSQLite directly. When enabled, the container generates a Litestream config file and execs Litestream with NSQLite as a child process.

### Enabling Litestream

| Variable                     | Description                                                                                       | Default |
| ---------------------------- | ------------------------------------------------------------------------------------------------- | ------- |
| `NSQLITE_LITESTREAM_ENABLED` | Set to `true`, `1`, `yes`, or `on` to enable Litestream replication. Any other value disables it. | `false` |

### S3 Replica (Required when Litestream is enabled)

| Variable                                  | Description                                                    | Example                              |
| ----------------------------------------- | -------------------------------------------------------------- | ------------------------------------ |
| `NSQLITE_LITESTREAM_S3_BUCKET`            | S3 bucket name for the replica destination.                    | `my-nsqlite-backups`                 |
| `NSQLITE_LITESTREAM_S3_PATH`              | Object key prefix (path) within the bucket.                    | `production/db/`                     |
| `NSQLITE_LITESTREAM_S3_ENDPOINT`          | S3-compatible endpoint URL. Use the full URL including scheme. | `https://s3.us-east-1.amazonaws.com` |
| `NSQLITE_LITESTREAM_S3_REGION`            | AWS region of the target bucket.                               | `us-east-1`                          |
| `NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID`     | AWS access key ID with write permission on the bucket.         | `AKIA...`                            |
| `NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY` | AWS secret access key corresponding to the access key ID.      | `...`                                |
| `NSQLITE_LITESTREAM_S3_SESSION_TOKEN`     | Optional AWS session token (for temporary credentials).        | `...`                                |

### Litestream Tuning

| Variable                                 | Description                                                                 | Default               |
| ---------------------------------------- | --------------------------------------------------------------------------- | --------------------- |
| `NSQLITE_LITESTREAM_CONFIG_PATH`         | Filesystem path where the auto-generated Litestream YAML config is written. | `/tmp/litestream.yml` |
| `NSQLITE_LITESTREAM_LOG_LEVEL`           | Litestream log level.                                                       | `info`                |
| `NSQLITE_LITESTREAM_LOG_FORMAT`          | Litestream log format (`text` or `json`).                                   | `text`                |
| `NSQLITE_LITESTREAM_SNAPSHOT_INTERVAL`   | Interval between full database snapshots.                                   | `24h`                 |
| `NSQLITE_LITESTREAM_SNAPSHOT_RETENTION`  | How long to retain old snapshots before pruning.                            | `168h` (7 days)       |
| `NSQLITE_LITESTREAM_SYNC_INTERVAL`       | How often Litestream syncs WAL changes to S3.                               | `1s`                  |
| `NSQLITE_LITESTREAM_VALIDATION_INTERVAL` | How often Litestream validates the integrity of the replica.                | `5m`                  |

> **Important:** When Litestream is enabled, the entrypoint writes the generated config file and then **execs Litestream**, which in turn spawns NSQLite as a managed child process. The `NSQLITE_DATA_DIR` variable controls where the database file is created (joined with `database.sqlite`).
