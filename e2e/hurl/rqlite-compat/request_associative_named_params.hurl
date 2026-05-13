POST {{host}}/db/execute
Content-Type: application/json
[BasicAuth]
ignored-user: {{admin_token}}
[
  "CREATE TABLE people (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"
]

HTTP 200
[Asserts]
jsonpath "$.results" count == 1

POST {{host}}/db/request?associative&timings
Content-Type: application/json
[BasicAuth]
ignored-user: {{rw_token}}
[
  ["INSERT INTO people(name, age) VALUES(:name, :age)", {"name": "declan", "age": 30}],
  ["SELECT name, age FROM people WHERE name=:name", {"name": "declan"}]
]

HTTP 200
[Asserts]
jsonpath "$.results" count == 2
jsonpath "$.results[0].rows_affected" == 1
jsonpath "$.results[1].types.name" == "text"
jsonpath "$.results[1].types.age" == "integer"
jsonpath "$.results[1].rows[0].name" == "declan"
jsonpath "$.results[1].rows[0].age" == 30
jsonpath "$.time" exists
