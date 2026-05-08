# Test INSERT ... RETURNING
POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE test_returning (id INTEGER PRIMARY KEY, name TEXT);"
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
      "query": "INSERT INTO test_returning (name) VALUES ('Jane') RETURNING *;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[0].columns[0]" == "id"
jsonpath "$.output.results[0].columns[1]" == "name"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[0].rows[0][1].text" == "Jane"

# Test queries with and without semicolons
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT 1, 2, 3;"
    },
    {
      "query": "SELECT 1, 2, 3"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[0].rows[0][1].integer" == 2
jsonpath "$.output.results[0].rows[0][2].integer" == 3
jsonpath "$.output.results[1].rows[0][0].integer" == 1
jsonpath "$.output.results[1].rows[0][1].integer" == 2
jsonpath "$.output.results[1].rows[0][2].integer" == 3

# Test structured errors
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT * FROM non_existent_table;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" != ""

# SQL execution errors stay inside the batch.
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT * FROM non_existent_table;"
    },
    {
      "query": "SELECT 42;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[1].type" == "read"
jsonpath "$.output.results[1].rows[0][0].integer" == 42
