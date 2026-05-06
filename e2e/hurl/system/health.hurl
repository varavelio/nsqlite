GET {{host}}/health
Authorization: Bearer {{admin_token}}

HTTP 200
[Asserts]
body == "OK"
