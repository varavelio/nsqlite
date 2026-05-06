POST {{host}}/query
Content-Type: application/json
```json
[
  {
    "query": "SELECT 1;"
  }
]
```
HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"
