package server

import (
	"net/http"

	"github.com/varavelio/nsqlite/internal/util/httputil"
	"github.com/varavelio/nsqlite/internal/version"
)

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) error {
	return httputil.WriteString(w, http.StatusOK, version.Version)
}
