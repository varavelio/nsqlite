GET {{host}}/version
Authorization: Bearer {{admin_token}}

HTTP 200
[Asserts]
body == "0.0.0-dev"
