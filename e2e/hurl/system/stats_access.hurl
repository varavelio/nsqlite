GET {{host}}/stats

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

GET {{host}}/stats
Authorization: Bearer invalid-token

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"

GET {{host}}/stats
Authorization: Bearer {{rw_token}}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"

GET {{host}}/stats
Authorization: Bearer {{ro_token}}

HTTP 403
[Asserts]
jsonpath "$.error" == "Forbidden"
jsonpath "$.message" == "Forbidden"
