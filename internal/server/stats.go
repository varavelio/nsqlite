package server

import (
	"time"

	"github.com/varavelio/nsqlite/internal/stats"
	"github.com/varavelio/nsqlite/internal/vdl"
	"github.com/varavelio/nsqlite/internal/version"
)

// systemSessionProc handles the System.session RPC procedure.
// It returns the authenticated role for the current request.
func (s *Server) systemSessionProc(
	c *vdl.SystemSessionHandlerContext[requestProps],
) (vdl.SystemSessionOutput, error) {
	return vdl.SystemSessionOutput{Role: authRoleToVDL(c.Props.Role)}, nil
}

// systemStatusProc handles the System.status RPC procedure.
// It returns server metadata, version information, and current database statistics.
func (s *Server) systemStatusProc(
	c *vdl.SystemStatusHandlerContext[requestProps],
) (vdl.SystemStatusOutput, error) {
	return vdl.SystemStatusOutput{
		Name:    "NSQLite",
		Version: version.Version,
		Stats:   loadedStatsToVDL(s.DBStats.LoadStats()),
	}, nil
}

// authRoleToVDL maps an internal auth role to the corresponding VDL type.
func authRoleToVDL(role authRole) vdl.AuthRole {
	switch role {
	case authRoleReadWrite:
		return vdl.AuthRoleReadWrite
	case authRoleReadOnly:
		return vdl.AuthRoleReadOnly
	default:
		return vdl.AuthRoleAdmin
	}
}

// loadedStatsToVDL converts internal loaded stats into the VDL representation.
func loadedStatsToVDL(loaded stats.LoadedStats) vdl.Stats {
	startedAt, err := time.Parse(time.RFC3339, loaded.StartedAt)
	if err != nil {
		startedAt = time.Time{}
	}

	minutes := make(map[string]vdl.StatsTotalsCounters, len(loaded.Stats))
	for _, minute := range loaded.Stats {
		minutes[minute.Minute] = vdl.StatsTotalsCounters{
			Reads:        minute.Reads,
			Writes:       minute.Writes,
			Begins:       minute.Begins,
			Commits:      minute.Commits,
			Rollbacks:    minute.Rollbacks,
			Errors:       minute.Errors,
			HttpRequests: minute.HTTPRequests,
		}
	}

	uptimeSeconds := 0.0
	if !startedAt.IsZero() {
		uptimeSeconds = time.Since(startedAt).Seconds()
	}

	return vdl.Stats{
		StartedAt:     startedAt,
		UptimeSeconds: uptimeSeconds,
		Totals: vdl.StatsTotalsCounters{
			Reads:        loaded.Totals.Reads,
			Writes:       loaded.Totals.Writes,
			Begins:       loaded.Totals.Begins,
			Commits:      loaded.Totals.Commits,
			Rollbacks:    loaded.Totals.Rollbacks,
			Errors:       loaded.Totals.Errors,
			HttpRequests: loaded.Totals.HTTPRequests,
		},
		Queued: vdl.StatsQueuedCounters{
			Begins:       loaded.QueuedBegins,
			Writes:       loaded.QueuedWrites,
			HttpRequests: loaded.QueuedHTTPRequests,
		},
		Minutes: minutes,
	}
}
