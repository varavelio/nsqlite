POST {{host}}/rpc/System/status
Content-Type: application/json
{}

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

POST {{host}}/rpc/System/status
Authorization: Bearer invalid-token
Content-Type: application/json
{}

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

POST {{host}}/rpc/System/status
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"

POST {{host}}/rpc/System/status
Authorization: Bearer {{ro_token}}
Content-Type: application/json
{}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"
