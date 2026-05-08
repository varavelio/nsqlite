POST {{host}}/rpc/System/status
Authorization: Bearer {{admin_token}}
Content-Type: application/json
{}

HTTP 200
[Asserts]
header "content-type" contains "application/json"
header "x-server" == "NSQLite"
jsonpath "$.ok" == true
jsonpath "$.output.stats.totals.reads" >= 1
jsonpath "$.output.stats.queued.begins" == 0
