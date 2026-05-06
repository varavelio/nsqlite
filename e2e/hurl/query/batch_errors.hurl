# Batch with error at the start — rest of the batch still executes
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT * FROM nonexistent_table;"
  },
  {
    "query": "SELECT 100;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" contains "no such table"
jsonpath "$.results[1].type" == "read"
jsonpath "$.results[1].rows[0][0]" == 100

# Batch with error in the middle — all queries are executed
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
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

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[1].type" == "error"
jsonpath "$.results[1].error" contains "no such table"
jsonpath "$.results[2].type" == "read"
jsonpath "$.results[2].rows[0][0]" == 2

# Batch with multiple errors interleaved with successes
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
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

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[1].type" == "error"
jsonpath "$.results[1].error" contains "no such table"
jsonpath "$.results[2].type" == "read"
jsonpath "$.results[3].type" == "error"
jsonpath "$.results[3].error" contains "no such table"
jsonpath "$.results[4].type" == "read"
jsonpath "$.results[4].rows[0][0]" == 30

# Empty query string produces a typed error result, rest of batch continues
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": ""
  },
  {
    "query": "SELECT 50;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "error"
jsonpath "$.results[0].error" == "Empty query"
jsonpath "$.results[1].type" == "read"
jsonpath "$.results[1].rows[0][0]" == 50

# Batch with failing CREATE TABLE followed by successful SELECT on existing table
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
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

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[1].type" == "error"
jsonpath "$.results[1].error" contains "already exists"
jsonpath "$.results[2].type" == "write"
jsonpath "$.results[3].type" == "read"
jsonpath "$.results[3].rows[0][0]" == 1
