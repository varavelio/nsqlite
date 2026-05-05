package nsqlited_tests

import (
	"net/http"
	"testing"

	"github.com/nsqlite/nsqlite/internal/nsqlite/config"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	queryToTest := Query{
		Query: "SELECT 1, 2, 3;",
	}

	tokenPlain := "some-auth-token"
	tokenBcrypt := "$2a$12$ydeSiOAMb4LSMfPwfiyjnemIE5iVSKIk9bNbCFcCWx75IWnhutGvG"
	tokenArgon2 := "$argon2id$v=19$m=16,t=2,p=1$c29tZS1hdXRoLXRva2Vu$stUgc57gBF5lQIpyk59xvQ"

	t.Run("Query without authentication", func(t *testing.T) {
		url := createServer(t) + "/query"
		res := sendQuery(t, url, queryToTest)
		assert.Equal(t, res.Results[0].Type, "read")
	})

	t.Run("Query sent without authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenPlain,
		})
		url += "/query"

		assertQueryStatus(t, url, "", queryToTest, http.StatusUnauthorized)
	})

	t.Run("Query sent with incorrect plain authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenPlain,
		})
		url += "/query"

		assertQueryStatus(t, url, "incorrect-token", queryToTest, http.StatusUnauthorized)
	})

	t.Run("Query sent with incorrect bcrypt authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenBcrypt,
		})
		url += "/query"

		assertQueryStatus(t, url, "incorrect-token", queryToTest, http.StatusUnauthorized)
	})

	t.Run("Query sent with incorrect argon2 authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenArgon2,
		})
		url += "/query"

		assertQueryStatus(t, url, "incorrect-token", queryToTest, http.StatusUnauthorized)
	})

	t.Run("Query sent with correct plain authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenPlain,
		})
		url += "/query"

		assertQueryStatus(t, url, tokenPlain, queryToTest, http.StatusOK)
	})

	t.Run("Query sent with correct bcrypt authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenBcrypt,
		})
		url += "/query"

		assertQueryStatus(t, url, tokenPlain, queryToTest, http.StatusOK)
	})

	t.Run("Query sent with correct argon2 authentication token", func(t *testing.T) {
		url := createServer(t, config.Config{
			AuthToken: tokenArgon2,
		})
		url += "/query"

		assertQueryStatus(t, url, tokenPlain, queryToTest, http.StatusOK)
	})
}
