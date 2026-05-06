GET {{host}}/health

HTTP 200
[Asserts]
header "content-type" contains "text/plain"
header "x-server" == "NSQLite"
body == "OK"
