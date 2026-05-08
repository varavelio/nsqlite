# Setup
POST {{host}}/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
[
  {
    "query": "CREATE TABLE reuse_test (id INTEGER PRIMARY KEY, val TEXT);"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"

# ── Scenario 1: BEGIN → INSERT → COMMIT → reuse same txId fails ──────────

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
tx_after_commit: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_commit}}",
    "query": "INSERT INTO reuse_test (id, val) VALUES (1, 'committed');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_commit}}",
    "query": "COMMIT;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "commit"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_commit}}",
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction not found"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_commit}}",
    "query": "COMMIT;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction not found"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_commit}}",
    "query": "ROLLBACK;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction not found"

# ── Scenario 2: BEGIN → INSERT → ROLLBACK → reuse same txId fails ────────

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
tx_after_rollback: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_rollback}}",
    "query": "INSERT INTO reuse_test (id, val) VALUES (2, 'rolled-back');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_rollback}}",
    "query": "ROLLBACK;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "rollback"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_after_rollback}}",
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction not found"

# ── Scenario 3: Fresh cycle works after COMMIT + new BEGIN ────────────────

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
tx_new: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_new}}",
    "query": "INSERT INTO reuse_test (id, val) VALUES (3, 'fresh-transaction');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_new}}",
    "query": "COMMIT;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "commit"

# Verify committed rows: row 1 exists, row 2 does not, row 3 exists
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT id, val FROM reuse_test ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[0].rows[0][1]" == "committed"
jsonpath "$.results[0].rows[1][0]" == 3
jsonpath "$.results[0].rows[1][1]" == "fresh-transaction"
jsonpath "$.results[0].rows" count == 2

# ── Scenario 4: COMMIT without begin on a fresh server connection still usable
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "COMMIT;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction ID is required"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "ROLLBACK;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "transaction ID is required"
