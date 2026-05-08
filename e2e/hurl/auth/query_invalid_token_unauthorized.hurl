POST {{host}}/rpc/Database/query
Authorization: Bearer invalid-token
Content-Type: application/json
{
  "queries": [
    {
      "query": "SELECT 1;"
    }
  ]
}

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"
