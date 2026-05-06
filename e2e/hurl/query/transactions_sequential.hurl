POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER NOT NULL, note TEXT NOT NULL);"
  },
  {
    "query": "INSERT INTO accounts (id, balance, note) VALUES (1, 10, 'seed');"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[1].type" == "write"

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
tx1: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx1}}",
    "query": "UPDATE accounts SET balance = balance + 5, note = 'tx1' WHERE id = 1;"
  },
  {
    "txId": "{{tx1}}",
    "query": "INSERT INTO accounts (id, balance, note) VALUES (2, 20, 'created-in-tx1');"
  },
  {
    "txId": "{{tx1}}",
    "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[1].type" == "write"
jsonpath "$.results[2].type" == "read"
jsonpath "$.results[2].rows[0][0]" == 1
jsonpath "$.results[2].rows[0][1]" == 15
jsonpath "$.results[2].rows[0][2]" == "tx1"
jsonpath "$.results[2].rows[1][0]" == 2
jsonpath "$.results[2].rows[1][1]" == 20
jsonpath "$.results[2].rows[1][2]" == "created-in-tx1"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[0].rows[0][1]" == 10
jsonpath "$.results[0].rows[0][2]" == "seed"
jsonpath "$.results[0].rows" count == 1

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx1}}",
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
    "query": "BEGIN;"
  }
]

HTTP 200
[Captures]
tx2: jsonpath "$.results[0].txId"
[Asserts]
jsonpath "$.results[0].type" == "begin"

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx2}}",
    "query": "UPDATE accounts SET balance = balance + 100 WHERE id = 1;"
  },
  {
    "txId": "{{tx2}}",
    "query": "DELETE FROM accounts WHERE id = 2;"
  },
  {
    "txId": "{{tx2}}",
    "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[1].type" == "write"
jsonpath "$.results[2].type" == "read"
jsonpath "$.results[2].rows[0][0]" == 1
jsonpath "$.results[2].rows[0][1]" == 115
jsonpath "$.results[2].rows[0][2]" == "tx1"
jsonpath "$.results[2].rows" count == 1

POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[
  {
    "txId": "{{tx2}}",
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
    "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1
jsonpath "$.results[0].rows[0][1]" == 15
jsonpath "$.results[0].rows[0][2]" == "tx1"
jsonpath "$.results[0].rows[1][0]" == 2
jsonpath "$.results[0].rows[1][1]" == 20
jsonpath "$.results[0].rows[1][2]" == "created-in-tx1"
jsonpath "$.results[0].rows" count == 2
