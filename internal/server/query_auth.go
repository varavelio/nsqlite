package server

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"

	"github.com/varavelio/nsqlite/internal/util/cryptoutil"
	"github.com/varavelio/nsqlite/internal/util/httputil"
)

// queryHandlerAuthMiddleware is a middleware that checks the Authorization
// header of the incoming request and compares it to the server's AuthToken
// configuration. If the AuthToken is empty, the middleware does nothing.
func (s *Server) queryHandlerAuthMiddleware(
	next httputil.HandlerFuncErr,
) httputil.HandlerFuncErr {
	return func(w http.ResponseWriter, r *http.Request) error {
		if s.AuthToken == "" {
			return next(w, r)
		}

		unauthorized := func() error {
			return httputil.NewJSONError(
				http.StatusUnauthorized, errors.New("Unauthorized"), "Unauthorized",
			)
		}

		clientAuthToken := r.Header.Get("Authorization")
		clientAuthToken = strings.TrimPrefix(clientAuthToken, "Bearer ")
		clientAuthToken = strings.TrimPrefix(clientAuthToken, "bearer ")
		if clientAuthToken == "" {
			return unauthorized()
		}

		if s.checkAuthWithCache(clientAuthToken) {
			return next(w, r)
		}

		return unauthorized()
	}
}

// checkAuthWithCache checks the client token against the in-memory cache
// first. On a cache hit it returns immediately; otherwise it runs the
// full checkAuthToken (Bcrypt/Argon2/plaintext) and caches the result on
// success.
func (s *Server) checkAuthWithCache(clientToken string) bool {
	if clientToken == "" {
		return false
	}

	hash := sha256.Sum256([]byte(s.authTokenSalt + clientToken))

	if _, ok := s.authTokenCache.Load(hash); ok {
		return true
	}

	if !checkAuthToken(s.authTokenAlgo, clientToken, s.AuthToken) {
		return false
	}

	s.authTokenCache.Store(hash, struct{}{})
	return true
}

// checkAuthToken checks if the token sent by the client matches the server's
// defined auth token.
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
