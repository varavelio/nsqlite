POST {{host}}/query
Authorization: {{rw_token}}
Content-Type: application/json
[
  {
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1
