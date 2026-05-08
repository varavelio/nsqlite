# Test INSERT ... RETURNING
POST {{host}}/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
[
  {
    "query": "CREATE TABLE test_returning (id INTEGER PRIMARY KEY, name TEXT);"
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
    "query": "INSERT INTO test_returning (name) VALUES ('Jane') RETURNING *;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[0].columns[0]" == "id"
jsonpath "$.results[0].columns[1]" == "name"
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[0].rows[0][1]" == "Jane"

# Test queries with and without semicolons
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT 1, 2, 3;"
  },
  {
    "query": "SELECT 1, 2, 3"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[0].rows[0][1]" == 2
jsonpath "$.results[0].rows[0][2]" == 3
jsonpath "$.results[1].rows[0][0]" == 1
jsonpath "$.results[1].rows[0][1]" == 2
jsonpath "$.results[1].rows[0][2]" == 3

# Test structured errors
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT * FROM non_existent_table;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" != ""

# SQL execution errors stay inside the batch.
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT * FROM non_existent_table;"
  },
  {
    "query": "SELECT 42;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[1].type" == "read"
jsonpath "$.results[1].rows[0][0]" == 42
