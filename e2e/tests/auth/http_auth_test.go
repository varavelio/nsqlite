package auth_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
	"github.com/varavelio/nsqlite/internal/util/cryptoutil"
)

func TestRoleTokensSupportPlaintextBcryptAndArgon2(t *testing.T) {
	t.Parallel()

	adminTokens := newRoleTokenSet(t, "admin")
	rwTokens := newRoleTokenSet(t, "rw")
	roTokens := newRoleTokenSet(t, "ro")

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   adminTokens.multiTokenServerConfigValue(),
		AuthTokenRW: rwTokens.multiTokenServerConfigValue(),
		AuthTokenRO: roTokens.multiTokenServerConfigValue(),
	})
	server.Query(t, adminTokens.plainClient, harness.Query{
		Query: "CREATE TABLE IF NOT EXISTS rw_access (id INTEGER PRIMARY KEY, name TEXT);",
	})

	t.Run(
		"admin plaintext bcrypt and argon2 tokens access admin endpoints and writes",
		func(t *testing.T) {
			for _, token := range adminTokens.multiTokenClientValues() {
				t.Run(token, func(t *testing.T) {
					stats := server.StatusResponse(t, token)
					require.Equal(t, http.StatusOK, stats.StatusCode)

					require.Equal(t, "0.0.0-dev", server.Version(t, token))

					response := server.Query(
						t,
						token,
						harness.Query{
							Query: "CREATE TABLE IF NOT EXISTS admin_access (id INTEGER PRIMARY KEY);",
						},
					)
					require.Equal(t, "write", response.Results[0].Type)
				})
			}
		},
	)

	t.Run(
		"read-write plaintext bcrypt and argon2 tokens can read and write but not access admin endpoints",
		func(t *testing.T) {
			for _, token := range rwTokens.multiTokenClientValues() {
				t.Run(token, func(t *testing.T) {
					readResponse := server.Query(t, token, harness.Query{Query: "SELECT 1;"})
					require.Equal(t, "read", readResponse.Results[0].Type)

					writeResponse := server.Query(
						t,
						token,
						harness.Query{
							Query: "INSERT INTO rw_access (name) VALUES ('created-by-rw');",
						},
					)
					require.Equal(t, "write", writeResponse.Results[0].Type)

					assertAPIError(
						t,
						server.StatusResponse(t, token),
						http.StatusOK,
						"Forbidden",
					)
				})
			}
		},
	)

	t.Run(
		"read-only plaintext bcrypt and argon2 tokens can read but not access admin endpoints",
		func(t *testing.T) {
			for _, token := range roTokens.multiTokenClientValues() {
				t.Run(token, func(t *testing.T) {
					response := server.Query(t, token, harness.Query{Query: "SELECT 1;"})
					require.Equal(t, "read", response.Results[0].Type)

					assertAPIError(
						t,
						server.StatusResponse(t, token),
						http.StatusOK,
						"Forbidden",
					)
				})
			}
		},
	)
}

func TestArgon2ConfiguredTokensAuthenticateAcrossRoles(t *testing.T) {
	t.Parallel()

	adminTokens := newRoleTokenSet(t, "admin")
	rwTokens := newRoleTokenSet(t, "rw")
	roTokens := newRoleTokenSet(t, "ro")

	for _, testCase := range []struct {
		name           string
		cfg            harness.ServerConfig
		request        func(t *testing.T, server *harness.Server) harness.HTTPResponse
		expectedStatus int
	}{
		{
			name: "admin argon2 token on stats",
			cfg:  harness.ServerConfig{AuthToken: adminTokens.argon2Hash},
			request: func(t *testing.T, server *harness.Server) harness.HTTPResponse {
				return server.StatusResponse(t, adminTokens.argon2Client)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "read-write argon2 token on query",
			cfg:  harness.ServerConfig{AuthTokenRW: rwTokens.argon2Hash},
			request: func(t *testing.T, server *harness.Server) harness.HTTPResponse {
				return server.QueryResponse(t, rwTokens.argon2Client, harness.Query{Query: "SELECT 1;"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "read-only argon2 token on query",
			cfg:  harness.ServerConfig{AuthTokenRO: roTokens.argon2Hash},
			request: func(t *testing.T, server *harness.Server) harness.HTTPResponse {
				return server.QueryResponse(t, roTokens.argon2Client, harness.Query{Query: "SELECT 1;"})
			},
			expectedStatus: http.StatusOK,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := harness.StartServer(t, testCase.cfg)
			response := testCase.request(t, server)
			require.Equal(
				t,
				testCase.expectedStatus,
				response.StatusCode,
				"unexpected response body: %s",
				string(response.Body),
			)
		})
	}
}

func TestSharedTokenRolePrecedenceIsObservable(t *testing.T) {
	t.Parallel()

	t.Run("admin token wins over read-write", func(t *testing.T) {
		server := harness.StartServer(t, harness.ServerConfig{
			AuthToken:   "shared-token",
			AuthTokenRW: "shared-token",
		})

		statsResponse := server.StatusResponse(t, "shared-token")
		require.Equal(t, http.StatusOK, statsResponse.StatusCode)

		writeResponse := server.Query(
			t,
			"shared-token",
			harness.Query{Query: "CREATE TABLE shared_admin (id INTEGER PRIMARY KEY);"},
		)
		require.Equal(t, "write", writeResponse.Results[0].Type)
	})

	t.Run("read-write token wins over read-only", func(t *testing.T) {
		server := harness.StartServer(t, harness.ServerConfig{
			AuthToken:   "admin-token",
			AuthTokenRW: "shared-token",
			AuthTokenRO: "shared-token",
		})
		server.Query(t, "admin-token", harness.Query{
			Query: "CREATE TABLE shared_rw (id INTEGER PRIMARY KEY, name TEXT);",
		})

		writeResponse := server.Query(
			t,
			"shared-token",
			harness.Query{Query: "INSERT INTO shared_rw (name) VALUES ('shared');"},
		)
		require.Equal(t, "write", writeResponse.Results[0].Type)

		assertAPIError(
			t,
			server.StatusResponse(t, "shared-token"),
			http.StatusOK,
			"Forbidden",
		)
	})
}

func TestReadOnlyTokenReceivesSQLiteAuthorizationErrorsForWriteAndTransactionQueries(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRO: "read-only-token",
	})
	server.Query(t, "admin-token", harness.Query{
		Query: "CREATE TABLE blocked_batch (id INTEGER PRIMARY KEY);",
	})

	readResponse := server.Query(t, "read-only-token", harness.Query{Query: "SELECT 1;"})
	require.Equal(t, harness.QueryResponse{
		Results: []harness.QueryResult{{
			Type:    "read",
			Columns: []string{"1"},
			Types:   []string{"INTEGER"},
			Rows:    [][]any{{float64(1)}},
		}},
	}, readResponse)

	for _, testCase := range []struct {
		name    string
		queries []harness.Query
	}{
		{
			name:    "write",
			queries: []harness.Query{{Query: "INSERT INTO blocked_batch DEFAULT VALUES;"}},
		},
		{
			name:    "begin",
			queries: []harness.Query{{Query: "BEGIN;"}},
		},
		{
			name: "mixed batch",
			queries: []harness.Query{
				{Query: "INSERT INTO blocked_batch DEFAULT VALUES;"},
				{Query: "SELECT 1;"},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := postJSONWithAuthorization(
				t,
				server,
				harness.DatabaseQueryPath,
				map[string]any{"queries": testCase.queries},
				"Bearer read-only-token",
			)
			require.Equal(t, http.StatusOK, response.StatusCode)
			queryResponse := harness.DecodeQueryResponse(t, response).WithoutTiming()
			require.Equal(t, "error", queryResponse.Results[0].Type)
			require.Contains(t, queryResponse.Results[0].Error, "23: authorization denied")

			if testCase.name == "mixed batch" {
				require.Len(t, queryResponse.Results, 2)
				require.Equal(t, "read", queryResponse.Results[1].Type)
			}
		})
	}

	for _, testCase := range []struct {
		name  string
		query string
	}{
		{name: "commit active transaction", query: "COMMIT;"},
		{name: "rollback active transaction", query: "ROLLBACK;"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			adminBegin := server.Query(t, "admin-token", harness.Query{Query: "BEGIN;"})
			txID := adminBegin.Results[0].TxID

			response := postJSONWithAuthorization(
				t,
				server,
				harness.DatabaseQueryPath,
				map[string]any{"queries": []harness.Query{{TxID: txID, Query: testCase.query}}},
				"Bearer read-only-token",
			)
			require.Equal(t, http.StatusOK, response.StatusCode)
			queryResponse := harness.DecodeQueryResponse(t, response).WithoutTiming()
			require.Equal(t, "error", queryResponse.Results[0].Type)
			require.Contains(t, queryResponse.Results[0].Error, "transaction ID does not match")

			cleanup := server.Query(t, "admin-token", harness.Query{Query: "ROLLBACK;", TxID: txID})
			require.Equal(t, "rollback", cleanup.Results[0].Type)
		})
	}

	tableCheck := server.Query(
		t,
		"read-only-token",
		harness.Query{
			Query: "SELECT COUNT(*) FROM blocked_batch;",
		},
	)
	require.Equal(t, [][]any{{float64(0)}}, tableCheck.Results[0].Rows)
}

type roleTokenSet struct {
	plainClient  string
	bcryptClient string
	argon2Client string
	bcryptHash   string
	argon2Hash   string
}

func newRoleTokenSet(t *testing.T, prefix string) roleTokenSet {
	t.Helper()

	bcryptClient := prefix + "-bcrypt"
	bcryptHash, err := cryptoutil.BcryptGenerateHash(bcryptClient)
	require.NoError(t, err)

	argon2Client := prefix + "-argon2"
	argon2Hash, err := cryptoutil.Argon2IDGenerateHash(argon2Client)
	require.NoError(t, err)

	return roleTokenSet{
		plainClient:  prefix + "-plain",
		bcryptClient: bcryptClient,
		argon2Client: argon2Client,
		bcryptHash:   bcryptHash,
		argon2Hash:   argon2Hash,
	}
}

func (s roleTokenSet) multiTokenClientValues() []string {
	return []string{s.plainClient, s.bcryptClient, s.argon2Client}
}

func (s roleTokenSet) multiTokenServerConfigValue() string {
	return strings.Join([]string{s.plainClient, s.bcryptHash, s.argon2Hash}, " ")
}

func postJSONWithAuthorization(
	t testing.TB,
	server *harness.Server,
	path string,
	body any,
	authorization string,
) harness.HTTPResponse {
	t.Helper()

	encodedBody, err := json.Marshal(body)
	require.NoError(t, err)

	return doRequest(t, server, http.MethodPost, path, bytes.NewReader(encodedBody), authorization)
}

func doRequest(
	t testing.TB,
	server *harness.Server,
	method, path string,
	body io.Reader,
	authorization string,
) harness.HTTPResponse {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, server.BaseURL()+path, body)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return harness.HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       responseBody,
	}
}

func assertAPIError(
	t testing.TB,
	response harness.HTTPResponse,
	expectedStatus int,
	expectedMessage string,
) {
	t.Helper()

	require.Equal(
		t,
		expectedStatus,
		response.StatusCode,
		"unexpected response body: %s",
		string(response.Body),
	)

	rpcError := harness.DecodeJSON[harness.RPCResponse[map[string]any]](t, response)
	require.False(t, rpcError.OK)
	require.Equal(t, expectedMessage, rpcError.Error.Message)
}
