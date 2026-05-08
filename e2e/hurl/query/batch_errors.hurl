# Batch with error at the start — rest of the batch still executes
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT * FROM nonexistent_table;"
    },
    {
      "query": "SELECT 100;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" contains "no such table"
jsonpath "$.output.results[1].type" == "read"
jsonpath "$.output.results[1].rows[0][0].integer" == 100

# Batch with error in the middle — all queries are executed
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT 1;"
    },
    {
      "query": "SELECT * FROM missing;"
    },
    {
      "query": "SELECT 2;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[1].type" == "error"
jsonpath "$.output.results[1].error" contains "no such table"
jsonpath "$.output.results[2].type" == "read"
jsonpath "$.output.results[2].rows[0][0].integer" == 2

# Batch with multiple errors interleaved with successes
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT 10;"
    },
    {
      "query": "SELECT * FROM unknown;"
    },
    {
      "query": "SELECT 20;"
    },
    {
      "query": "SELECT * FROM also_missing;"
    },
    {
      "query": "SELECT 30;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[1].type" == "error"
jsonpath "$.output.results[1].error" contains "no such table"
jsonpath "$.output.results[2].type" == "read"
jsonpath "$.output.results[3].type" == "error"
jsonpath "$.output.results[3].error" contains "no such table"
jsonpath "$.output.results[4].type" == "read"
jsonpath "$.output.results[4].rows[0][0].integer" == 30

# Empty query string produces a typed error result, rest of batch continues
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": ""
    },
    {
      "query": "SELECT 50;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "error"
jsonpath "$.output.results[0].error" == "Empty query"
jsonpath "$.output.results[1].type" == "read"
jsonpath "$.output.results[1].rows[0][0].integer" == 50

# Batch with failing CREATE TABLE followed by successful SELECT on existing table
POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE existing (id INTEGER PRIMARY KEY);"
    },
    {
      "query": "CREATE TABLE existing (id INTEGER PRIMARY KEY);"
    },
    {
      "query": "INSERT INTO existing (id) VALUES (1);"
    },
    {
      "query": "SELECT id FROM existing;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[1].type" == "error"
jsonpath "$.output.results[1].error" contains "already exists"
jsonpath "$.output.results[2].type" == "write"
jsonpath "$.output.results[3].type" == "read"
jsonpath "$.output.results[3].rows[0][0].integer" == 1
