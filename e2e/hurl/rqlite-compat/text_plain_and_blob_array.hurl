POST {{host}}/db/execute
Content-Type: text/plain
[BasicAuth]
ignored-user: {{admin_token}}
`CREATE TABLE blobs (id INTEGER PRIMARY KEY, data BLOB)`

HTTP 200
[Asserts]
jsonpath "$.results" count == 1

POST {{host}}/db/execute
Content-Type: application/json
[BasicAuth]
ignored-user: {{rw_token}}
[
  ["INSERT INTO blobs(data) VALUES(?)", [83, 81, 76, 105, 116, 101]],
  ["INSERT INTO blobs(data) VALUES(?)", "x'68656C6C6F'"]
]

HTTP 200
[Asserts]
jsonpath "$.results[0].rows_affected" == 1
jsonpath "$.results[1].rows_affected" == 1

GET {{host}}/db/query?blob_array&q=SELECT%20data%20FROM%20blobs
[BasicAuth]
ignored-user: {{ro_token}}

HTTP 200
[Asserts]
jsonpath "$.results[0].columns[0]" == "data"
jsonpath "$.results[0].types[0]" == "blob"
jsonpath "$.results[0].values[0][0][0]" == 83
jsonpath "$.results[0].values[0][0][1]" == 81
jsonpath "$.results[0].values[0][0][5]" == 101
jsonpath "$.results[0].values[1][0][0]" == 104
jsonpath "$.results[0].values[1][0][4]" == 111
