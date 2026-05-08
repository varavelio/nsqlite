POST {{host}}/rpc/Database/query
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{
  "queries": [
    {
      "query": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"
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
      "query": "INSERT INTO users (name) VALUES ('Ada');"
    },
    {
      "query": "SELECT id, name FROM users;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "write"
jsonpath "$.output.results[0].lastInsertId" == 1
jsonpath "$.output.results[1].type" == "read"
jsonpath "$.output.results[1].columns[0]" == "id"
jsonpath "$.output.results[1].columns[1]" == "name"
jsonpath "$.output.results[1].rows[0][0].integer" == 1
jsonpath "$.output.results[1].rows[0][1].text" == "Ada"
