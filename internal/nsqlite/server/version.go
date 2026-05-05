package server

import (
	"net/http"

	"github.com/nsqlite/nsqlite/internal/util/httputil"
	"github.com/nsqlite/nsqlite/internal/version"
)

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) error {
	return httputil.WriteString(w, http.StatusOK, version.Version)
}
