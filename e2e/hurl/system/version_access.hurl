POST {{host}}/rpc/System/status
Content-Type: application/json
{}

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Unauthorized"

POST {{host}}/rpc/System/status
Authorization: Bearer invalid-token
Content-Type: application/json
{}

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Unauthorized"

POST {{host}}/rpc/System/status
Authorization: Bearer {{rw_token}}
Content-Type: application/json
{}

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Forbidden"

POST {{host}}/rpc/System/status
Authorization: Bearer {{ro_token}}
Content-Type: application/json
{}

HTTP 200
[Asserts]
jsonpath "$.ok" == false
jsonpath "$.error.message" == "Forbidden"
