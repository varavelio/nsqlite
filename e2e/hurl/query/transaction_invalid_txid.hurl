# Wrong txIds are rejected before classification while the database remains usable.

POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE parent (id TEXT PRIMARY KEY);"
    },
    {
      "query": "CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent(id));"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[1].type" == "write"

# ── Scenario 1: wrong txId during an active transaction ───────────────────

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
active_tx: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"
jsonpath "$.output.results[0].error" not exists

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{active_tx}}",
      "query": "INSERT INTO parent (id) VALUES ('p1');"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[0].error" not exists

# This statement is only valid while the uncommitted parent row is visible from
# the active transaction. With a wrong txId, it must fail with the txId mismatch
# error instead of a foreign key or preparation error.
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "wrong-active-transaction-id",
      "query": "INSERT INTO child (id, parent_id) VALUES ('c1', 'p1');"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" == "transaction ID does not match the currently active transaction"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{active_tx}}",
      "query": "ROLLBACK;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "rollback"
jsonpath "$.output.results[0].error" not exists

# ── Scenario 2: txId when no transaction exists ───────────────────────────

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "tx-id-with-no-active-transaction",
      "query": "SELECT 1;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" == "transaction not found or timed out, check your settings"

# The server and database should still respond normally after both txId errors.
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT 1;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].error" not exists
jsonpath "$.output.results[0].rows[0][0].integer" == 1
