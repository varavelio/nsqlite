POST {{host}}/rpc/System/status
Authorization: Bearer invalid-token
Content-Type: application/json
{}

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"
