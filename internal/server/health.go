package server

import (
	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/vdl"
)

func (s *Server) systemHealthProc(
	c *vdl.SystemHealthHandlerContext[requestProps],
) (vdl.SystemHealthOutput, error) {
	_, err := s.DB.Query(c.Context, db.Query{Query: "SELECT 1"})
	if err != nil {
		s.Logger.Error(c.Context, "health check failed", "error", err.Error())
		return vdl.SystemHealthOutput{
			Healthy:  false,
			Database: false,
			Message:  "Database unavailable",
		}, nil
	}

	return vdl.SystemHealthOutput{
		Healthy:  true,
		Database: true,
		Message:  "OK",
	}, nil
}
