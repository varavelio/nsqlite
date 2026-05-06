GET {{host}}/stats
Authorization: Bearer {{admin_token}}

HTTP 200
[Asserts]
header "content-type" contains "application/json"
header "x-server" == "NSQLite"
jsonpath "$.totals.reads" >= 1
jsonpath "$.queuedBegins" == 0
