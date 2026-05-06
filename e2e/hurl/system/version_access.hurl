GET {{host}}/version

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

GET {{host}}/version
Authorization: Bearer invalid-token

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

GET {{host}}/version
Authorization: Bearer {{rw_token}}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"

GET {{host}}/version
Authorization: Bearer {{ro_token}}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"
