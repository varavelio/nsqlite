POST {{host}}/rpc/System/status
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{}

HTTP 200
[Asserts]
header "content-type" contains "application/json"
jsonpath "$.ok" == true
jsonpath "$.output.version" == "0.0.0-dev"
