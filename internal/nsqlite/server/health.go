package server

import (
	"net/http"

	"github.com/nsqlite/nsqlite/internal/nsqlite/db"
	"github.com/nsqlite/nsqlite/internal/util/httputil"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) error {
	_, err := s.DB.Query(r.Context(), db.Query{
		Query: "SELECT 1",
	})
	if err != nil {
		return httputil.NewJSONError(
			http.StatusInternalServerError, err, "Failed to query the database",
		)
	}

	return httputil.WriteString(w, http.StatusOK, "OK")
}
