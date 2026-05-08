# NSQLite API Reference

Base URL: `http://<host>:<port>` (default port: `9876`)

All responses include the header `X-Server: NSQLite`.

---

## Authentication

Endpoints marked **🔒 Admin** require a valid `Authorization: Bearer <token>` header with an admin-level token.

Endpoints marked **🔒 Query** require a valid `Authorization: Bearer <token>` header. The role (admin, read/write, or read-only) determines which SQL operations are allowed — see the [`/query`](#post-query) endpoint for details.

If no authentication tokens are configured (`NSQLITE_AUTH_TOKEN`, `NSQLITE_AUTH_TOKEN_RW`, and `NSQLITE_AUTH_TOKEN_RO` are all empty), the server runs in **unauthenticated mode** and all requests are treated as admin.

**Error format** (all error responses):

```json
{
  "id": "a1b2c3d4-...",
  "error": "Unauthorized",
  "message": "Unauthorized"
}
```

- `id` — Unique error identifier (UUID v4), useful for cross-referencing server logs.
- `error` — HTTP status text.
- `message` — Human-readable description of the error.

---

## Endpoints

### `GET /health` — Health Check

**Auth:** None

**Description:** Verifies the server is alive and can successfully execute a simple SQL query (`SELECT 1`) against the underlying SQLite database.

**Response** `200 OK`:

```
OK
```

**Response** `500 Internal Server Error` — returned when the database is unreachable or the query fails.

---

### `GET /version` — Version Info

**Auth:** 🔒 Admin

**Description:** Returns the current NSQLite version string.

**Response** `200 OK`:

```
v0.1.0
```

_(Version format depends on the build; plain text body.)_

---

### `GET /stats` — Server Statistics

**Auth:** 🔒 Admin

**Description:** Returns live server and database statistics. The exact fields depend on the stats implementation but typically include HTTP request counters, queue depth, and database connection pool metrics.

**Response** `200 OK`:

```json
{
  "httpRequests": 42,
  "httpRequestsQueued": 3,
  "httpRequestsQueuedMax": 10,
  "dbTotalTime": 1.234,
  "dbTotalTimeMax": 0.567,
  "dbAveTime": 0.029
}
```

**Response Fields** (subject to change based on build):

| Field                   | Description                                                    |
| ----------------------- | -------------------------------------------------------------- |
| `httpRequests`          | Total number of HTTP requests handled since server start.      |
| `httpRequestsQueued`    | Currently queued HTTP requests awaiting a database connection. |
| `httpRequestsQueuedMax` | Maximum number of requests ever queued simultaneously.         |
| `dbTotalTime`           | Cumulative time (in seconds) spent on database operations.     |
| `dbTotalTimeMax`        | Longest single database operation time (in seconds).           |
| `dbAveTime`             | Average database operation time (in seconds).                  |

---

### `POST /query` — Execute SQL Queries

**Auth:** 🔒 Query (Admin = all operations, Read/Write = queries + transactions, Read-Only = read queries only)

**Description:** Submit one or more SQL statements for execution. The endpoint supports batching — send an array of queries in a single request.

**Request:**

```json
[
  {
    "query": "SELECT * FROM users WHERE id = ?",
    "params": [1],
    "txId": ""
  },
  {
    "query": "INSERT INTO users (name) VALUES (?)",
    "params": ["Alice"],
    "txId": ""
  }
]
```

**Request Fields:**

| Field    | Type     | Required | Description                                                                                      |
| -------- | -------- | -------- | ------------------------------------------------------------------------------------------------ |
| `query`  | `string` | ✅       | The SQL statement to execute.                                                                    |
| `params` | `array`  | ❌       | Parameters for parameterized queries (`?` placeholders).                                         |
| `txId`   | `string` | ❌       | Transaction ID obtained from a `BEGIN` query. Omit or leave empty for non-transactional queries. |

**Response** `200 OK`:

```json
{
  "time": 0.0023,
  "results": [
    {
      "type": "read",
      "time": 0.0011,
      "columns": ["id", "name"],
      "types": ["integer", "text"],
      "rows": [[1, "Alice"]]
    },
    {
      "type": "write",
      "time": 0.0012,
      "lastInsertId": 2,
      "rowsAffected": 1,
      "columns": [],
      "types": [],
      "rows": []
    }
  ]
}
```

**Response Fields (top-level):**

| Field     | Type    | Description                                                 |
| --------- | ------- | ----------------------------------------------------------- |
| `time`    | `float` | Total execution time for all queries in seconds.            |
| `results` | `array` | Array of per-query results, in the same order as the input. |

**Per-Result Fields (`results[]`):**

| Field          | Type       | Applies To       | Description                                                              |
| -------------- | ---------- | ---------------- | ------------------------------------------------------------------------ |
| `type`         | `string`   | All              | Result type: `read`, `write`, `begin`, `commit`, `rollback`, or `error`. |
| `time`         | `float`    | All              | Execution time for this single query in seconds.                         |
| `error`        | `string`   | `error` results  | Error message if the query failed.                                       |
| `txId`         | `string`   | `begin` results  | Transaction ID to use in subsequent queries within the same transaction. |
| `lastInsertId` | `int`      | `write` results  | The `rowid` of the last inserted row.                                    |
| `rowsAffected` | `int`      | `write` results  | Number of rows modified by the statement.                                |
| `columns`      | `[string]` | `read` / `write` | Column names of the result set.                                          |
| `types`        | `[string]` | `read` / `write` | SQLite column type affinity names.                                       |
| `rows`         | `[array]`  | `read` / `write` | Result set rows, each row is an ordered array of values.                 |

**Query Types:**

| Type       | Description                                     | Allowed Roles                    |
| ---------- | ----------------------------------------------- | -------------------------------- |
| `read`     | SELECT, EXPLAIN, etc.                           | Admin, Read/Write, **Read-Only** |
| `write`    | INSERT, UPDATE, DELETE, DDL                     | Admin, Read/Write                |
| `begin`    | Start a transaction (`BEGIN`), returns a `txId` | Admin, Read/Write                |
| `commit`   | Commit a transaction (`COMMIT`)                 | Admin, Read/Write                |
| `rollback` | Rollback a transaction (`ROLLBACK`)             | Admin, Read/Write                |

**Transaction Handling:**

1. Send `BEGIN` as a query — the response includes a `txId`.
2. Include that `txId` in subsequent query objects within the same request **or** across separate requests.
3. Send `COMMIT` or `ROLLBACK` with the same `txId` to finish the transaction.

> **Idle timeout:** If no activity occurs on the transaction within `NSQLITE_TX_IDLE_TIMEOUT` (default `10s`), it is automatically rolled back.

---

### Error Responses (All Endpoints)

| Status Code                    | Meaning                                                                                                     |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| `400 Bad Request`              | Malformed request body (e.g., invalid JSON, empty query).                                                   |
| `401 Unauthorized`             | Missing or invalid `Authorization` header.                                                                  |
| `403 Forbidden`                | Authenticated but the token's role does not permit the requested operation.                                 |
| `413 Request Entity Too Large` | Request body exceeds `NSQLITE_MAX_REQUEST_SIZE_MB`.                                                         |
| `500 Internal Server Error`    | Unexpected server error. The response body contains a unique error ID that can be found in the server logs. |
