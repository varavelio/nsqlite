GET {{host}}/version
Authorization: Bearer {{admin_token}}

HTTP 200
[Asserts]
header "content-type" contains "text/plain"
header "x-server" == "NSQLite"
body == "0.0.0-dev"
