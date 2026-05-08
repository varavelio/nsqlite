# Setup
POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE reuse_test (id INTEGER PRIMARY KEY, val TEXT);"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"

# ── Scenario 1: BEGIN → INSERT → COMMIT → reuse same txId fails ──────────

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "BEGIN;"
    }
  ]
}

HTTP 200
[Captures]
tx_after_commit: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_commit}}",
      "query": "INSERT INTO reuse_test (id, val) VALUES (1, 'committed');"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_commit}}",
      "query": "COMMIT;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "commit"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_commit}}",
      "query": "SELECT 1;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction not found"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_commit}}",
      "query": "COMMIT;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction not found"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_commit}}",
      "query": "ROLLBACK;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction not found"

# ── Scenario 2: BEGIN → INSERT → ROLLBACK → reuse same txId fails ────────

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "BEGIN;"
    }
  ]
}

HTTP 200
[Captures]
tx_after_rollback: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_rollback}}",
      "query": "INSERT INTO reuse_test (id, val) VALUES (2, 'rolled-back');"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_rollback}}",
      "query": "ROLLBACK;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "rollback"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_after_rollback}}",
      "query": "SELECT 1;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction not found"

# ── Scenario 3: Fresh cycle works after COMMIT + new BEGIN ────────────────

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "BEGIN;"
    }
  ]
}

HTTP 200
[Captures]
tx_new: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_new}}",
      "query": "INSERT INTO reuse_test (id, val) VALUES (3, 'fresh-transaction');"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx_new}}",
      "query": "COMMIT;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "commit"

# Verify committed rows: row 1 exists, row 2 does not, row 3 exists
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT id, val FROM reuse_test ORDER BY id;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[0].rows[0][1].text" == "committed"
jsonpath "$.output.results[0].rows[1][0].integer" == 3
jsonpath "$.output.results[0].rows[1][1].text" == "fresh-transaction"
jsonpath "$.output.results[0].rows" count == 2

# ── Scenario 4: COMMIT without begin on a fresh server connection still usable
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "COMMIT;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction ID is required"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "ROLLBACK;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "transaction ID is required"
