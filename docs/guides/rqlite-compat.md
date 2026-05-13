# RQLite API Compatibility

NSQLite provides an additive compatibility layer for clients that use the [rqlite HTTP API](https://rqlite.io/docs/api/api/) format. This layer translates rqlite-style requests into NSQLite's native query execution path and translates responses back to the rqlite JSON shape.

The native NSQLite RPC API remains unchanged and is the preferred way to use NSQLite.

## Routes

| Method | Route         | Purpose                                      |
| ------ | ------------- | -------------------------------------------- |
| `GET`  | `/db/query`   | Execute read queries from the `q` parameter. |
| `POST` | `/db/query`   | Execute read queries from the request body.  |
| `POST` | `/db/execute` | Execute write statements.                    |
| `POST` | `/db/request` | Execute mixed read/write statements.         |

## Authentication

The compatibility routes support NSQLite Bearer tokens and rqlite-style Basic auth.

When Basic auth is used, NSQLite ignores the username and treats the password as the NSQLite token. For example, `ignored-user:rw-token` authenticates with `rw-token`.

## Request Formats

JSON statement arrays are supported:

```json
[
  "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
  ["INSERT INTO users(name) VALUES(?)", "fiona"]
]
```

Named parameters are supported:

```json
[
  ["INSERT INTO users(name) VALUES(:name)", { "name": "fiona" }]
]
```

Plain-text single statements are supported for `POST` requests with `Content-Type: text/plain`:

```sql
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)
```

## Query Parameters

| Parameter     | Behavior                                                                      |
| ------------- | ----------------------------------------------------------------------------- |
| `q`           | SQL query for `GET /db/query`.                                                |
| `transaction` | Executes all statements in one transaction and rolls back on the first error. |
| `timings`     | Includes per-result and total response timings.                               |
| `associative` | Returns read rows as objects keyed by column name.                            |
| `blob_array`  | Returns BLOB values as byte arrays instead of base64 strings.                 |

Other rqlite cluster-specific parameters, such as `redirect`, `retries`, `raft_index`, and read-consistency controls, are accepted as no-ops because NSQLite is not a Raft cluster.

## Response Shape

Responses follow the rqlite `results` shape:

```json
{
  "results": [
    {
      "last_insert_id": 1,
      "rows_affected": 1
    },
    {
      "columns": ["id", "name"],
      "types": ["integer", "text"],
      "values": [[1, "fiona"]]
    }
  ]
}
```

Database-level errors are returned inside individual result objects using the rqlite `error` key while preserving HTTP `200`, matching rqlite's documented behavior.
