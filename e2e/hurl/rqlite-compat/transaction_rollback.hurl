POST {{host}}/db/execute
Content-Type: application/json
[BasicAuth]
ignored-user: {{admin_token}}
[
  "CREATE TABLE rollback_items (id INTEGER PRIMARY KEY, name TEXT)"
]

HTTP 200
[Asserts]
jsonpath "$.results" count == 1

POST {{host}}/db/execute?transaction
Content-Type: application/json
[BasicAuth]
ignored-user: {{rw_token}}
[
  ["INSERT INTO rollback_items(id, name) VALUES(?, ?)", 1, "first"],
  ["INSERT INTO rollback_items(id, name) VALUES(?, ?)", 1, "duplicate"]
]

HTTP 200
[Asserts]
jsonpath "$.results" count == 2
jsonpath "$.results[0].rows_affected" == 1
jsonpath "$.results[1].error" contains "constraint"

GET {{host}}/db/query?q=SELECT%20COUNT(*)%20FROM%20rollback_items
[BasicAuth]
ignored-user: {{ro_token}}

HTTP 200
[Asserts]
jsonpath "$.results[0].values[0][0]" == 0
