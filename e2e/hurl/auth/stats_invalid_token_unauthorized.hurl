GET {{host}}/stats
Authorization: Bearer invalid-token

HTTP 401
[Asserts]
jsonpath "$.error" == "Unauthorized"
jsonpath "$.message" == "Unauthorized"
