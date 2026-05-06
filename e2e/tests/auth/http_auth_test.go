package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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

	t.Run(
		"admin plaintext bcrypt and argon2 tokens access admin endpoints and writes",
		func(t *testing.T) {
			for _, token := range adminTokens.multiTokenClientValues() {
				t.Run(token, func(t *testing.T) {
					stats := server.Get(t, "/stats", token)
					require.Equal(t, http.StatusOK, stats.StatusCode)

					version := server.Get(t, "/version", token)
					require.Equal(t, http.StatusOK, version.StatusCode)
					require.Equal(t, "0.0.0-dev", string(version.Body))

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
							Query: "CREATE TABLE IF NOT EXISTS rw_access (id INTEGER PRIMARY KEY);",
						},
					)
					require.Equal(t, "write", writeResponse.Results[0].Type)

					assertAPIError(
						t,
						server.Get(t, "/stats", token),
						http.StatusForbidden,
						"Forbidden",
					)
					assertAPIError(
						t,
						server.Get(t, "/version", token),
						http.StatusForbidden,
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
						server.Get(t, "/stats", token),
						http.StatusForbidden,
						"Forbidden",
					)
					assertAPIError(
						t,
						server.Get(t, "/version", token),
						http.StatusForbidden,
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
				return server.Get(t, "/stats", adminTokens.argon2Client)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "read-write argon2 token on query",
			cfg:  harness.ServerConfig{AuthTokenRW: rwTokens.argon2Hash},
			request: func(t *testing.T, server *harness.Server) harness.HTTPResponse {
				return server.PostJSON(t, "/query", []harness.Query{{Query: "SELECT 1;"}}, rwTokens.argon2Client)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "read-only argon2 token on query",
			cfg:  harness.ServerConfig{AuthTokenRO: roTokens.argon2Hash},
			request: func(t *testing.T, server *harness.Server) harness.HTTPResponse {
				return server.PostJSON(t, "/query", []harness.Query{{Query: "SELECT 1;"}}, roTokens.argon2Client)
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

		statsResponse := server.Get(t, "/stats", "shared-token")
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
			AuthTokenRW: "shared-token",
			AuthTokenRO: "shared-token",
		})

		writeResponse := server.Query(
			t,
			"shared-token",
			harness.Query{Query: "CREATE TABLE shared_rw (id INTEGER PRIMARY KEY);"},
		)
		require.Equal(t, "write", writeResponse.Results[0].Type)

		assertAPIError(
			t,
			server.Get(t, "/stats", "shared-token"),
			http.StatusForbidden,
			"Forbidden",
		)
	})
}

func TestReadOnlyTokenRejectsWriteTransactionAndMixedBatchOperations(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthTokenRO: "read-only-token",
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
			queries: []harness.Query{{Query: "CREATE TABLE blocked (id INTEGER PRIMARY KEY);"}},
		},
		{
			name:    "begin",
			queries: []harness.Query{{Query: "BEGIN;"}},
		},
		{
			name:    "commit",
			queries: []harness.Query{{Query: "COMMIT;"}},
		},
		{
			name:    "rollback",
			queries: []harness.Query{{Query: "ROLLBACK;"}},
		},
		{
			name: "mixed batch",
			queries: []harness.Query{
				{Query: "CREATE TABLE blocked_batch (id INTEGER PRIMARY KEY);"},
				{Query: "SELECT 1;"},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := postJSONWithAuthorization(
				t,
				server,
				"/query",
				testCase.queries,
				"Bearer read-only-token",
			)
			assertAPIError(t, response, http.StatusForbidden, "Forbidden")
		})
	}

	tableCheck := server.Query(
		t,
		"read-only-token",
		harness.Query{
			Query: "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'blocked_batch';",
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

	apiError := harness.DecodeJSON[harness.APIError](t, response)
	require.Equal(t, expectedMessage, apiError.Error)
	require.Equal(t, expectedMessage, apiError.Message)
	require.NotEmpty(t, apiError.ID)
}
