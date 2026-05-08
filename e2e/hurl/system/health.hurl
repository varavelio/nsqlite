POST {{host}}/rpc/System/health
Content-Type: application/json
{}

HTTP 200
[Asserts]
header "content-type" contains "application/json"
jsonpath "$.ok" == true
jsonpath "$.output.healthy" == true
jsonpath "$.output.database" == true
jsonpath "$.output.message" == "OK"
