<p align="center">
  <img width="300" src="https://cdn.jsdelivr.net/gh/varavelio/nsqlite@dbf7ff/assets/logo.svg" alt="NSQLite logo"/>
</p>

<p align="center">SQLite over the network.</p>

<p align="center">
  <a href="https://github.com/varavelio/nsqlite/actions/workflows/ci.yaml?query=branch%3Amain">
    <img src="https://github.com/varavelio/nsqlite/actions/workflows/ci.yaml/badge.svg" alt="CI Status"/>
  </a>
  <a href="https://goreportcard.com/report/varavelio/nsqlite">
    <img src="https://goreportcard.com/badge/varavelio/nsqlite" alt="Go Report Card"/>
  </a>
  <a href="https://github.com/varavelio/nsqlite/releases/latest">
    <img src="https://img.shields.io/github/release/varavelio/nsqlite.svg" alt="Release Version"/>
  </a>
  <a href="https://hub.docker.com/r/varavel/nsqlite">
    <img alt="Docker Hub Pulls" src="https://img.shields.io/docker/pulls/varavel/nsqlite?label=docker%20hub%20pulls"/>
  </a>
  <a href="https://github.com/orgs/varavelio/packages/container/package/nsqlite">
    <img alt="GHCR Pulls" src="https://img.shields.io/badge/ghcr%20pulls-package%20stats-blue?logo=github"/>
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/varavelio/nsqlite.svg" alt="License"/>
  </a>
  <a href="https://github.com/varavelio/nsqlite">
    <img src="https://img.shields.io/github/stars/varavelio/nsqlite?style=flat&label=github+stars"/>
  </a>
</p>

<p align="center">
  <a href="https://varavel.com">
    <img src="https://cdn.jsdelivr.net/gh/varavelio/brand@1.0.0/dist/badges/project.svg" alt="A Varavel project"/>
  </a>
</p>

## Overview

`nsqlite` runs a SQLite-backed HTTP server and ships with a container image that can optionally wrap the process with `litestream` for continuous replication to any S3-compatible object store.

Container images are published to both Docker Hub and GitHub Container Registry with matching tags:

- Docker Hub: `varavel/nsqlite:<tag>`
- GHCR: `ghcr.io/varavelio/nsqlite:<tag>`

## Quick Start

Run NSQLite without Litestream:

```bash
docker run --rm -p 9876:9876 -v ./data:/data varavel/nsqlite:latest
```

You can also use the GHCR mirror by replacing the image with `ghcr.io/varavelio/nsqlite:latest`.

Run NSQLite with Litestream and an S3-compatible replica:

```bash
docker run --rm \
  -p 9876:9876 \
  -v ./data:/data \
  -e NSQLITE_LITESTREAM_ENABLED=true \
  -e NSQLITE_LITESTREAM_S3_BUCKET="my-backups" \
  -e NSQLITE_LITESTREAM_S3_PATH="db-backup/database.sqlite" \
  -e NSQLITE_LITESTREAM_S3_ENDPOINT="https://minio.example.com:9000" \
  -e NSQLITE_LITESTREAM_S3_REGION="us-east-1" \
  -e NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID="your-access-key" \
  -e NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY="your-secret-key" \
  varavel/nsqlite:latest
```

## NSQLite Configuration

The container always configures NSQLite through environment variables and it has sane default values.

| Variable                      | Container default | Description                                                                                                                  |
| ----------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `NSQLITE_AUTH_TOKEN`          | unset             | Admin token list. Use space-separated plaintext tokens or bcrypt/argon2id hashes for full access.                            |
| `NSQLITE_AUTH_TOKEN_RW`       | unset             | Read/write token list. Use space-separated plaintext tokens or bcrypt/argon2id hashes for query read/write access only.      |
| `NSQLITE_AUTH_TOKEN_RO`       | unset             | Read-only token list. Use space-separated plaintext tokens or bcrypt/argon2id hashes for query read access only.             |
| `NSQLITE_DATA_DIR`            | `/data`           | Directory used by NSQLite to store its SQLite files. The main database file is always `${NSQLITE_DATA_DIR}/database.sqlite`. |
| `NSQLITE_LISTEN_HOST`         | `0.0.0.0`         | Host/interface NSQLite binds to inside the container.                                                                        |
| `NSQLITE_LISTEN_PORT`         | `9876`            | TCP port used by the HTTP server.                                                                                            |
| `NSQLITE_TX_IDLE_TIMEOUT`     | `10s`             | Maximum idle time for an open transaction before it is rolled back.                                                          |
| `NSQLITE_MAX_READ_CONNS`      | `10`              | Maximum number of read-only SQLite connections.                                                                              |
| `NSQLITE_CACHE_SIZE_KB`       | `20000`           | SQLite cache size in KB per connection.                                                                                      |
| `NSQLITE_BUSY_TIMEOUT`        | `5s`              | How long SQLite waits when the database is locked by another writer.                                                         |
| `NSQLITE_MAX_REQUEST_SIZE_MB` | `100`             | Maximum HTTP body size accepted by the `/query` endpoint.                                                                    |

> **⚠️ Important: Auth tokens security**
>
> Always prefer using token hashes over plaintext tokens. The recommended algorithm is **Argon2ID**, followed by **Bcrypt** as a secondary option. Plaintext tokens are discouraged, as they can be exposed if environment variables or the container are compromised.

## Litestream Configuration

When `NSQLITE_LITESTREAM_ENABLED=true`, the container writes a Litestream config that uses the explicit S3-compatible configurations.

Credentials are intentionally not written to the generated YAML file. The entrypoint translates the S3 credential variables into the runtime environment that Litestream expects.

### Required Litestream Variables

These variables are required when `NSQLITE_LITESTREAM_ENABLED=true`.

| Variable                                  | Description                                                                                                                |
| ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `NSQLITE_LITESTREAM_ENABLED`              | Set to `true` to run NSQLite under Litestream.                                                                             |
| `NSQLITE_LITESTREAM_S3_BUCKET`            | Bucket or container name used for the replica.                                                                             |
| `NSQLITE_LITESTREAM_S3_PATH`              | Object path inside the bucket for the replica data.                                                                        |
| `NSQLITE_LITESTREAM_S3_ENDPOINT`          | S3-compatible endpoint. Examples: `s3.us-east-1.wasabisys.com`, `https://minio.example.com:9000`, `http://localhost:9000`. |
| `NSQLITE_LITESTREAM_S3_REGION`            | Replica region. Use the provider value that matches the target bucket.                                                     |
| `NSQLITE_LITESTREAM_S3_ACCESS_KEY_ID`     | Access key used by Litestream.                                                                                             |
| `NSQLITE_LITESTREAM_S3_SECRET_ACCESS_KEY` | Secret key used by Litestream.                                                                                             |

### Optional Litestream Variables

| Variable                                 | Default               | Description                                                      |
| ---------------------------------------- | --------------------- | ---------------------------------------------------------------- |
| `NSQLITE_LITESTREAM_S3_SESSION_TOKEN`    | unset                 | Optional session token when temporary credentials are used.      |
| `NSQLITE_LITESTREAM_CONFIG_PATH`         | `/tmp/litestream.yml` | Where the container writes the generated Litestream config file. |
| `NSQLITE_LITESTREAM_LOG_LEVEL`           | `info`                | Litestream log level written into the generated config.          |
| `NSQLITE_LITESTREAM_LOG_FORMAT`          | `text`                | Litestream log format written into the generated config.         |
| `NSQLITE_LITESTREAM_SNAPSHOT_INTERVAL`   | `24h`                 | Snapshot creation interval.                                      |
| `NSQLITE_LITESTREAM_SNAPSHOT_RETENTION`  | `168h`                | Snapshot retention period.                                       |
| `NSQLITE_LITESTREAM_SYNC_INTERVAL`       | `1s`                  | How often Litestream syncs changes to the replica.               |
| `NSQLITE_LITESTREAM_VALIDATION_INTERVAL` | `5m`                  | How often Litestream validates replica state.                    |

## Notes for S3-Compatible Providers

- If your provider accepts an endpoint without a scheme, you can pass it directly. Litestream assumes HTTPS by default.
- For local development, an explicit `http://` endpoint is valid.
- The endpoint and region must match the target provider and bucket configuration.
- This image keeps the configuration surface intentionally small. Advanced provider-specific settings should be added only when there is a concrete need.

## License

This project is released under the MIT License. See [LICENSE](LICENSE).
