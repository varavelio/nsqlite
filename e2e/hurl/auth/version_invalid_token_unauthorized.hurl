GET {{host}}/version
Authorization: Bearer invalid-token

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"
