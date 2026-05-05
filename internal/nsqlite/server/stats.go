package server

import (
	"net/http"

	"github.com/varavelio/nsqlite/internal/util/httputil"
)

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) error {
	stats := s.DBStats.LoadStats()
	return httputil.WriteJSON(w, http.StatusOK, stats)
}
