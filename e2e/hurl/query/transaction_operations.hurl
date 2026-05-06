# Setup table for all transaction scenarios
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "CREATE TABLE txn_items (id INTEGER PRIMARY KEY, val TEXT NOT NULL);"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"

# ── Scenario 1: BEGIN → INSERT → ROLLBACK discards all changes ──────────

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
tx_rollback: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_rollback}}",
    "query": "INSERT INTO txn_items (id, val) VALUES (1, 'discard-me');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[0].lastInsertId" == 1
jsonpath "$.results[0].rowsAffected" == 1

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_rollback}}",
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
    "query": "SELECT COUNT(*) FROM txn_items;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 0

# ── Scenario 2: BEGIN → INSERT valid → INSERT duplicate PK → COMMIT ────
# The valid INSERT persists and COMMIT succeeds after the constraint error.

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
tx_pk: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_pk}}",
    "query": "INSERT INTO txn_items (id, val) VALUES (10, 'valid');"
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
    "txId": "{{tx_pk}}",
    "query": "INSERT INTO txn_items (id, val) VALUES (10, 'duplicate-pk');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_pk}}",
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
    "query": "SELECT id, val FROM txn_items ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 10
jsonpath "$.results[0].rows[0][1]" == "valid"
jsonpath "$.results[0].rows" count == 1

# ── Scenario 3: BEGIN → INSERT valid → INSERT syntax error → COMMIT ────
# A syntax error in a write does NOT prevent the COMMIT of valid operations.

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
tx_syntax: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_syntax}}",
    "query": "INSERT INTO txn_items (id, val) VALUES (20, 'before-syntax-error');"
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
    "txId": "{{tx_syntax}}",
    "query": "INSERT INTO txn_items (id, val) VALUES (,,,);"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx_syntax}}",
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
    "query": "SELECT id, val FROM txn_items WHERE id = 20;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 20
jsonpath "$.results[0].rows[0][1]" == "before-syntax-error"
