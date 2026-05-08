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

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Unauthorized"
