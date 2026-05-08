POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
{
  "queries": [
    {
      "query": "SELECT 1;"
    }
  ]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1

# Invalid JSON returns an RPC envelope error.
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
```
{
  "queries": [
    {
      "query": "SELECT 1;"
    }
  ]
```

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Invalid request body"

# Unknown fields are ignored.
POST {{host}}/rpc/Database/query
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{
  "queries": [{
    "query": "SELECT 2;",
    "unknown": "ignored",
    "metadata": {
      "nested": true
    }
  }]
}

HTTP 200
[Asserts]
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 2
