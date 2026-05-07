# Wrong txIds are rejected before classification while the database remains usable.

# ── Scenario 1: wrong txId during an active transaction ───────────────────

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "BEGIN;"
  }
]

HTTP 200
[Captures]
active_tx: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"
jsonpath "$.results[0].error" not exists

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{active_tx}}",
    "query": "CREATE TABLE parent (id TEXT PRIMARY KEY);"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[0].error" not exists

# This statement would be valid only from the active transaction's schema view.
# With a wrong txId, it must fail with the txId mismatch error instead of a
# SQLite classification/preparation error.
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "wrong-active-transaction-id",
    "query": "CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent(id));"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" == "transaction ID does not match the currently active transaction"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{active_tx}}",
    "query": "ROLLBACK;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "rollback"
jsonpath "$.results[0].error" not exists

# ── Scenario 2: txId when no transaction exists ───────────────────────────

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "tx-id-with-no-active-transaction",
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" == "transaction not found or timed out, check your settings"

# The server and database should still respond normally after both txId errors.
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].error" not exists
jsonpath "$.results[0].rows[0][0]" == 1
