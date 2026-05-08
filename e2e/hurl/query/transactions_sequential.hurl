POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE accounts (id INTEGER PRIMARY KEY, balance INTEGER NOT NULL, note TEXT NOT NULL);"
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
      "query": "INSERT INTO accounts (id, balance, note) VALUES (1, 10, 'seed');"
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
      "query": "BEGIN;"
    }
  ]
}

HTTP 200
[Captures]
tx1: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
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
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[1].type" == "write"
jsonpath "$.output.results[2].type" == "read"
jsonpath "$.output.results[2].rows[0][0].integer" == 1
jsonpath "$.output.results[2].rows[0][1].integer" == 15
jsonpath "$.output.results[2].rows[0][2].text" == "tx1"
jsonpath "$.output.results[2].rows[1][0].integer" == 2
jsonpath "$.output.results[2].rows[1][1].integer" == 20
jsonpath "$.output.results[2].rows[1][2].text" == "created-in-tx1"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[0].rows[0][1].integer" == 10
jsonpath "$.output.results[0].rows[0][2].text" == "seed"
jsonpath "$.output.results[0].rows" count == 1

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx1}}",
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
      "query": "BEGIN;"
    }
  ]
}

HTTP 200
[Captures]
tx2: jsonpath "$.output.results[0].txId"
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "begin"

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
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
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[1].type" == "write"
jsonpath "$.output.results[2].type" == "read"
jsonpath "$.output.results[2].rows[0][0].integer" == 1
jsonpath "$.output.results[2].rows[0][1].integer" == 115
jsonpath "$.output.results[2].rows[0][2].text" == "tx1"
jsonpath "$.output.results[2].rows" count == 1

POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [
    {
      "txId": "{{tx2}}",
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
      "query": "SELECT id, balance, note FROM accounts ORDER BY id;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
jsonpath "$.output.results[0].rows[0][1].integer" == 15
jsonpath "$.output.results[0].rows[0][2].text" == "tx1"
jsonpath "$.output.results[0].rows[1][0].integer" == 2
jsonpath "$.output.results[0].rows[1][1].integer" == 20
jsonpath "$.output.results[0].rows[1][2].text" == "created-in-tx1"
jsonpath "$.output.results[0].rows" count == 2
