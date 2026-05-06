package nsqlited_tests

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/varavelio/nsqlite/internal/config"
)

func TestAuth(t *testing.T) {
	queryToTest := Query{Query: "SELECT 1, 2, 3;"}
	writeQuery := Query{Query: "CREATE TABLE test (id INTEGER PRIMARY KEY);"}

	adminPlain := "admin-plain"
	adminBcrypt := "$2a$12$ydeSiOAMb4LSMfPwfiyjnemIE5iVSKIk9bNbCFcCWx75IWnhutGvG"
	adminArgon2 := "$argon2id$v=19$m=16,t=2,p=1$c29tZS1hdXRoLXRva2Vu$stUgc57gBF5lQIpyk59xvQ"
	rwPlain := "rw-plain"
	rwPlain2 := "rw-plain-2"
	roPlain := "ro-plain"
	roPlain2 := "ro-plain-2"

	baseURL := createServer(t, config.Config{
		AuthToken:   adminPlain + "," + adminBcrypt + "," + adminArgon2,
		AuthTokenRW: rwPlain + "," + rwPlain2,
		AuthTokenRO: roPlain + "," + roPlain2,
	})

	t.Run("health works without authentication", func(t *testing.T) {
		url := baseURL + "/health"

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		assert.NoError(t, err)
		res, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() { _ = res.Body.Close() }()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, "OK", string(body))
	})

	t.Run("query without token is unauthorized", func(t *testing.T) {
		assertQueryStatus(t, baseURL+"/query", "", queryToTest, http.StatusUnauthorized)
	})

	t.Run("multiple admin tokens support plaintext bcrypt and argon2", func(t *testing.T) {
		assertQueryStatus(t, baseURL+"/query", adminPlain, queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", "some-auth-token", queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", "some-auth-token", queryToTest, http.StatusOK)
	})

	t.Run("read write token can read and write query endpoint", func(t *testing.T) {
		assertQueryStatus(t, baseURL+"/query", rwPlain, queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", rwPlain2, queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", rwPlain, writeQuery, http.StatusOK)
	})

	t.Run("read only token can read but cannot write query endpoint", func(t *testing.T) {
		assertQueryStatus(t, baseURL+"/query", roPlain, queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", roPlain2, queryToTest, http.StatusOK)
		assertQueryStatus(t, baseURL+"/query", roPlain, writeQuery, http.StatusForbidden)
	})

	t.Run("read only token cannot use transaction control queries", func(t *testing.T) {
		assertQueryStatus(t, baseURL+"/query", roPlain, Query{Query: "BEGIN"}, http.StatusForbidden)
	})

	t.Run("admin only endpoints reject read write and read only tokens", func(t *testing.T) {
		for _, endpoint := range []string{"/stats", "/version"} {
			for _, token := range []string{rwPlain, roPlain} {
				req, err := http.NewRequestWithContext(
					context.Background(),
					http.MethodGet,
					baseURL+endpoint,
					nil,
				)
				assert.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)

				res, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				_ = res.Body.Close()

				assert.Equal(t, http.StatusForbidden, res.StatusCode)
			}
		}
	})

	t.Run("admin token can access stats", func(t *testing.T) {
		stats := getStatsWithToken(t, baseURL, adminPlain)
		assert.GreaterOrEqual(t, stats.Totals.Reads, int64(1))
	})
}
