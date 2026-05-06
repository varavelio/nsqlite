package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/util/cryptoutil"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

type authRole string

const (
	authRoleAdmin     authRole = "admin"
	authRoleReadWrite authRole = "read-write"
	authRoleReadOnly  authRole = "read-only"
)

type authToken struct {
	role  authRole
	algo  cryptoutil.HashAlgo
	value string
}

type contextKey string

const (
	queryContextKey contextKey = "query"
)

func newAuthTokens(adminTokens, readWriteTokens, readOnlyTokens []string) []authToken {
	tokens := make([]authToken, 0, len(adminTokens)+len(readWriteTokens)+len(readOnlyTokens))
	for _, token := range adminTokens {
		tokens = append(
			tokens,
			authToken{role: authRoleAdmin, algo: cryptoutil.GetHashAlgo(token), value: token},
		)
	}
	for _, token := range readWriteTokens {
		tokens = append(
			tokens,
			authToken{role: authRoleReadWrite, algo: cryptoutil.GetHashAlgo(token), value: token},
		)
	}
	for _, token := range readOnlyTokens {
		tokens = append(
			tokens,
			authToken{role: authRoleReadOnly, algo: cryptoutil.GetHashAlgo(token), value: token},
		)
	}
	return tokens
}

// adminAuthMiddleware allows only admin tokens.
func (s *Server) adminAuthMiddleware(next httputil.HandlerFuncErr) httputil.HandlerFuncErr {
	return func(w http.ResponseWriter, r *http.Request) error {
		role, err := s.authenticateRequest(r)
		if err != nil {
			return err
		}
		if role != authRoleAdmin {
			return forbiddenError()
		}
		return next(w, r)
	}
}

// queryHandlerAuthMiddleware authenticates the request and enforces query permissions.
func (s *Server) queryHandlerAuthMiddleware(next httputil.HandlerFuncErr) httputil.HandlerFuncErr {
	return func(w http.ResponseWriter, r *http.Request) error {
		role, err := s.authenticateRequest(r)
		if err != nil {
			return err
		}

		var queries []Query
		if err := json.NewDecoder(r.Body).Decode(&queries); err != nil {
			return httputil.NewJSONError(
				http.StatusBadRequest, err, "Failed to read request body",
			)
		}

		for _, query := range queries {
			if query.Query == "" {
				continue
			}

			if role != authRoleReadOnly {
				continue
			}

			queryType, err := s.DB.ClassifyQuery(r.Context(), query.Query)
			if err != nil {
				return forbiddenError()
			}

			if !isQueryAllowed(role, queryType) {
				return forbiddenError()
			}
		}

		ctx := context.WithValue(r.Context(), queryContextKey, queries)
		return next(w, r.WithContext(ctx))
	}
}

func isQueryAllowed(role authRole, queryType db.QueryType) bool {
	switch role {
	case authRoleAdmin:
		return true
	case authRoleReadWrite:
		return queryType == db.QueryTypeRead ||
			queryType == db.QueryTypeWrite ||
			queryType == db.QueryTypeBegin ||
			queryType == db.QueryTypeCommit ||
			queryType == db.QueryTypeRollback
	case authRoleReadOnly:
		return queryType == db.QueryTypeRead
	default:
		return false
	}
}

func (s *Server) authenticateRequest(r *http.Request) (authRole, error) {
	clientAuthToken := r.Header.Get("Authorization")
	clientAuthToken = strings.TrimPrefix(clientAuthToken, "Bearer ")
	clientAuthToken = strings.TrimPrefix(clientAuthToken, "bearer ")
	if clientAuthToken == "" {
		return "", unauthorizedError()
	}

	role, ok := s.checkAuthWithCache(clientAuthToken)
	if !ok {
		return "", unauthorizedError()
	}

	return role, nil
}

// checkAuthWithCache checks the client token against the in-memory cache first.
// On a cache hit it returns immediately; otherwise it runs the full auth check
// (bcrypt/argon2/plaintext) and caches the role on success.
func (s *Server) checkAuthWithCache(clientToken string) (authRole, bool) {
	if clientToken == "" {
		return "", false
	}

	hash := sha256.Sum256([]byte(s.authTokenSalt + clientToken))

	if cachedRole, ok := s.authTokenCache.Load(hash); ok {
		role, ok := cachedRole.(authRole)
		return role, ok
	}

	for _, token := range s.authTokens {
		if !checkAuthToken(token.algo, clientToken, token.value) {
			continue
		}

		s.authTokenCache.Store(hash, token.role)
		return token.role, true
	}

	return "", false
}

// checkAuthToken checks if the token sent by the client matches the server token.
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
		http.StatusUnauthorized, errors.New("Unauthorized"), "Unauthorized",
	)
}

func forbiddenError() error {
	return httputil.NewJSONError(
		http.StatusForbidden, errors.New("Forbidden"), "Forbidden",
	)
}

func getQueriesFromContext(ctx context.Context) ([]Query, bool) {
	queries, ok := ctx.Value(queryContextKey).([]Query)
	return queries, ok
}
