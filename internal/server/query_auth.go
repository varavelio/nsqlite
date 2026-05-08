package server

import (
	"context"
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

// contextKey is the private key type used for request context values.
type contextKey string

const (
	authRoleContextKey      contextKey = "auth-role"
	authPrincipalContextKey contextKey = "auth-principal"
)

// newAuthTokens builds the in-memory auth token list for all configured roles.
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

// adminAuthMiddleware allows only admin requests when authentication is enabled.
func (s *Server) adminAuthMiddleware(next httputil.HandlerFuncErr) httputil.HandlerFuncErr {
	return func(w http.ResponseWriter, r *http.Request) error {
		role, _, err := s.authenticateRequest(r)
		if err != nil {
			return err
		}

		if role != authRoleAdmin {
			return forbiddenError()
		}

		return next(w, r)
	}
}

// queryHandlerAuthMiddleware authenticates the request and stores its role and
// principal identity in the request context.
func (s *Server) queryHandlerAuthMiddleware(next httputil.HandlerFuncErr) httputil.HandlerFuncErr {
	return func(w http.ResponseWriter, r *http.Request) error {
		role, principal, err := s.authenticateRequest(r)
		if err != nil {
			return err
		}

		ctx := context.WithValue(r.Context(), authRoleContextKey, role)
		ctx = context.WithValue(ctx, authPrincipalContextKey, principal)
		return next(w, r.WithContext(ctx))
	}
}

// authenticateRequest authenticates the incoming request and resolves its role
// together with a stable principal identity derived from the presented token.
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
// On a cache hit it returns immediately; otherwise it runs the full auth check
// (bcrypt/argon2/plaintext) and caches the resolved role and principal on success.
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

// checkAuthToken reports whether the client token matches the configured server token.
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

// unauthorizedError returns the standard unauthorized API error.
func unauthorizedError() error {
	return httputil.NewJSONError(
		http.StatusUnauthorized,
		errors.New("Unauthorized"),
		"Unauthorized",
	)
}

// forbiddenError returns the standard forbidden API error.
func forbiddenError() error {
	return httputil.NewJSONError(
		http.StatusForbidden,
		errors.New("Forbidden"),
		"Forbidden",
	)
}

// getAuthRoleFromContext reads the authenticated role from the request context.
func getAuthRoleFromContext(ctx context.Context) (authRole, bool) {
	role, ok := ctx.Value(authRoleContextKey).(authRole)
	return role, ok
}

// getAuthPrincipalFromContext reads the authenticated principal identity from
// the request context.
func getAuthPrincipalFromContext(ctx context.Context) (string, bool) {
	principal, ok := ctx.Value(authPrincipalContextKey).(string)
	return principal, ok
}
