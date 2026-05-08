POST {{host}}/rpc/System/status
Authorization: Bearer invalid-token
Content-Type: application/json
{}

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Unauthorized"
