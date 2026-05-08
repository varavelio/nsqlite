POST {{host}}/rpc/Database/query
Authorization: bearer {{rw_token}}
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
jsonpath "$.ok" == true
jsonpath "$.output.results[0].type" == "read"
jsonpath "$.output.results[0].rows[0][0].integer" == 1
