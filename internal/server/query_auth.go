package server

import (
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

		if checkAuthToken(s.authTokenAlgo, clientAuthToken, s.AuthToken) {
			return next(w, r)
		}

		return unauthorized()
	}
}

// checkAuthToken checks if the token sent by the client matches the server's
// defined auth token.
func checkAuthToken(tokenAlgo cryptoutil.HashAlgo, clientToken, serverToken string) bool {
	if tokenAlgo == cryptoutil.HashAlgoPlaintext {
		return clientToken == serverToken
	}

	if tokenAlgo == cryptoutil.HashAlgoArgon2 {
		return cryptoutil.Argon2CheckHash(clientToken, serverToken)
	}

	if tokenAlgo == cryptoutil.HashAlgoBcrypt {
		return cryptoutil.BcryptCheckHash(clientToken, serverToken)
	}

	return false
}
