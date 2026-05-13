POST {{host}}/db/execute?timings
Content-Type: application/json
[BasicAuth]
ignored-user: {{admin_token}}
[
  "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)",
  ["INSERT INTO users(name, age) VALUES(?, ?)", "fiona", 20]
]

HTTP 200
[Asserts]
jsonpath "$.results" count == 2
jsonpath "$.results[1].last_insert_id" == 1
jsonpath "$.results[1].rows_affected" == 1
jsonpath "$.time" exists

GET {{host}}/db/query?timings&q=SELECT%20id%2C%20name%2C%20age%20FROM%20users
[BasicAuth]
ignored-user: {{ro_token}}

HTTP 200
[Asserts]
jsonpath "$.results" count == 1
jsonpath "$.results[0].columns[0]" == "id"
jsonpath "$.results[0].columns[1]" == "name"
jsonpath "$.results[0].columns[2]" == "age"
jsonpath "$.results[0].types[0]" == "integer"
jsonpath "$.results[0].types[1]" == "text"
jsonpath "$.results[0].types[2]" == "integer"
jsonpath "$.results[0].values[0][0]" == 1
jsonpath "$.results[0].values[0][1]" == "fiona"
jsonpath "$.results[0].values[0][2]" == 20
