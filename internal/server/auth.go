package server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/varavelio/nsqlite/internal/util/cryptoutil"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

// authRole identifies the authorization level granted to a request.
type authRole string

const (
	authRoleAdmin     authRole = "admin"
	authRoleReadWrite authRole = "read-write"
	authRoleReadOnly  authRole = "read-only"
)

// authToken stores a configured token together with its resolved role and hash algorithm.
type authToken struct {
	role  authRole
	algo  cryptoutil.HashAlgo
	value string
}

// newAuthTokens builds the in-memory auth token list from all configured token sets.
func newAuthTokens(adminTokens, readWriteTokens, readOnlyTokens []string) []authToken {
	tokens := make([]authToken, 0, len(adminTokens)+len(readWriteTokens)+len(readOnlyTokens))

	for _, token := range adminTokens {
		tokens = append(tokens, authToken{
			role:  authRoleAdmin,
			algo:  cryptoutil.GetHashAlgo(token),
			value: token,
		})
	}

	for _, token := range readWriteTokens {
		tokens = append(tokens, authToken{
			role:  authRoleReadWrite,
			algo:  cryptoutil.GetHashAlgo(token),
			value: token,
		})
	}

	for _, token := range readOnlyTokens {
		tokens = append(tokens, authToken{
			role:  authRoleReadOnly,
			algo:  cryptoutil.GetHashAlgo(token),
			value: token,
		})
	}

	return tokens
}

// authenticateRequest extracts and verifies the Bearer token from the request.
// It returns the resolved auth role and a stable principal identity derived from the token.
func (s *Server) authenticateRequest(r *http.Request) (authRole, string, error) {
	if s.authIsDisabled() {
		return authRoleAdmin, "", nil
	}

	clientAuthToken := r.Header.Get("Authorization")
	clientAuthToken = strings.TrimPrefix(clientAuthToken, "Bearer ")
	clientAuthToken = strings.TrimPrefix(clientAuthToken, "bearer ")
	if clientAuthToken == "" {
		return "", "", unauthorizedError()
	}

	role, principal, ok := s.checkAuthWithCache(clientAuthToken)
	if !ok {
		return "", "", unauthorizedError()
	}

	return role, principal, nil
}

// checkAuthWithCache checks the client token against the in-memory cache first.
// On a cache miss it scans the configured tokens (bcrypt/argon2id/plaintext)
// and caches the resolved role on success.
func (s *Server) checkAuthWithCache(clientToken string) (authRole, string, bool) {
	if clientToken == "" {
		return "", "", false
	}

	hash := sha256.Sum256([]byte(s.authTokenSalt + clientToken))
	principal := fmt.Sprintf("%x", hash)

	if cachedRole, ok := s.authTokenCache.Load(hash); ok {
		role, ok := cachedRole.(authRole)
		return role, principal, ok
	}

	for _, token := range s.authTokens {
		if !checkAuthToken(token.algo, clientToken, token.value) {
			continue
		}

		s.authTokenCache.Store(hash, token.role)
		return token.role, principal, true
	}

	return "", "", false
}

// checkAuthToken reports whether the client token matches the configured server token
// using the appropriate hash comparison for the token's algorithm.
func checkAuthToken(tokenAlgo cryptoutil.HashAlgo, clientToken, serverToken string) bool {
	if tokenAlgo == cryptoutil.HashAlgoPlaintext {
		return clientToken == serverToken
	}

	if tokenAlgo == cryptoutil.HashAlgoArgon2ID {
		return cryptoutil.Argon2IDCheckHash(clientToken, serverToken)
	}

	if tokenAlgo == cryptoutil.HashAlgoBcrypt {
		return cryptoutil.BcryptCheckHash(clientToken, serverToken)
	}

	return false
}

func unauthorizedError() error {
	return httputil.NewJSONError(
		http.StatusUnauthorized,
		errors.New("Unauthorized"),
		"Unauthorized",
	)
}

func forbiddenError() error {
	return httputil.NewJSONError(
		http.StatusForbidden,
		errors.New("Forbidden"),
		"Forbidden",
	)
}
