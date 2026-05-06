package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestOpenServerAllowsAdminEndpointsAndWritesWithoutTokens(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})

	versionResponse := server.Get(t, "/version", "")
	require.Equal(t, http.StatusOK, versionResponse.StatusCode)
	require.Equal(t, "0.0.0-dev", string(versionResponse.Body))

	stats := server.Stats(t, "")
	require.GreaterOrEqual(t, stats.Totals.Reads, int64(1))

	writeResponse := server.Query(
		t,
		"",
		harness.Query{Query: "CREATE TABLE test (id INTEGER PRIMARY KEY);"},
	)
	require.Equal(t, "write", writeResponse.Results[0].Type)
}

func TestMultipleAdminTokensSupportPlaintextAndBcrypt(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken: "admin-plain $2a$12$ydeSiOAMb4LSMfPwfiyjnemIE5iVSKIk9bNbCFcCWx75IWnhutGvG",
	})

	for _, token := range []string{"admin-plain", "some-auth-token"} {
		t.Run(token, func(t *testing.T) {
			response := server.Query(t, token, harness.Query{Query: "SELECT 1, 2, 3;"})
			require.Equal(t, "read", response.Results[0].Type)
		})
	}
}

func TestReadWriteTokenCanReadAndWrite(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{AuthTokenRW: "rw-plain rw-plain-2"})

	for _, token := range []string{"rw-plain", "rw-plain-2"} {
		t.Run(token, func(t *testing.T) {
			readResponse := server.Query(t, token, harness.Query{Query: "SELECT 1;"})
			require.Equal(t, "read", readResponse.Results[0].Type)
		})
	}

	writeResponse := server.Query(
		t,
		"rw-plain",
		harness.Query{Query: "CREATE TABLE test (id INTEGER PRIMARY KEY);"},
	)
	require.Equal(t, "write", writeResponse.Results[0].Type)
}

func TestAdminOnlyEndpointsRejectReadWriteAndReadOnlyTokens(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		AuthToken:   "admin-token",
		AuthTokenRW: "rw-token",
		AuthTokenRO: "ro-token",
	})

	for _, endpoint := range []string{"/stats", "/version"} {
		for _, token := range []string{"rw-token", "ro-token"} {
			t.Run(endpoint+" "+token, func(t *testing.T) {
				response := server.Get(t, endpoint, token)
				require.Equal(t, http.StatusForbidden, response.StatusCode)

				apiError := harness.DecodeJSON[harness.APIError](t, response)
				require.Equal(t, "Forbidden", apiError.Error)
				require.Equal(t, "Forbidden", apiError.Message)
				require.NotEmpty(t, apiError.ID)
			})
		}
	}
}

func TestAdminTokenCanAccessStats(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{AuthToken: "admin-token"})
	stats := server.Stats(t, "admin-token")

	require.GreaterOrEqual(t, stats.Totals.Reads, int64(1))
}
