POST {{host}}/query
Authorization: Bearer {{rw_token}}
[
  {
    "query": "SELECT 1;"
  }
]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 1

# Invalid JSON returns a 400 API error.
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
```
[
  {
    "query": "SELECT 1;"
  }
```

HTTP 400
[Asserts]
jsonpath "$.error" == "Bad Request"
jsonpath "$.message" == "Failed to read request body"

# Unknown fields are ignored.
POST {{host}}/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
[{
  "query": "SELECT 2;",
  "unknown": "ignored",
  "metadata": {
    "nested": true
  }
}]

HTTP 200
[Asserts]
jsonpath "$.results[0].type" == "read"
jsonpath "$.results[0].rows[0][0]" == 2
