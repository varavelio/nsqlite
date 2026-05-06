POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
```json
[
  {
    "query": "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"
  },
  {
    "query": "INSERT INTO users (name) VALUES ('Ada');"
  },
  {
    "query": "SELECT id, name FROM users;"
  }
]
```
HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "write"
jsonpath "$.results[1].type" == "write"
jsonpath "$.results[1].lastInsertId" == 1
jsonpath "$.results[2].type" == "read"
jsonpath "$.results[2].columns[0]" == "id"
jsonpath "$.results[2].columns[1]" == "name"
jsonpath "$.results[2].rows[0][0]" == 1
jsonpath "$.results[2].rows[0][1]" == "Ada"
