GET {{host}}/db/query?q=SELECT%201

HTTP 401
[Asserts]
jsonpath "$.message" == "Unauthorized"

GET {{host}}/db/query?q=SELECT%201
[BasicAuth]
any-user-is-ignored: wrong-token

HTTP 401
[Asserts]
jsonpath "$.message" == "Unauthorized"

GET {{host}}/db/query?q=SELECT%201
[BasicAuth]
any-user-is-ignored: {{rw_token}}

HTTP 200
[Asserts]
jsonpath "$.results[0].values[0][0]" == 1
