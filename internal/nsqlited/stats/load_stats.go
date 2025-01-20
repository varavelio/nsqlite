package stats

import (
	"sort"
	"time"
)

type LoadedStats struct {
	StartedAt          string `json:"startedAt"`
	Uptime             string `json:"uptime"`
	QueuedBegins       int64  `json:"queuedBegins"`
	QueuedWrites       int64  `json:"queuedWrites"`
	QueuedHTTPRequests int64  `json:"queuedHttpRequests"`
	Totals             Totals `json:"totals"`
	Stats              []Stat `json:"stats"`
}

type Totals struct {
	Reads        int64 `json:"reads"`
	Writes       int64 `json:"writes"`
	Begins       int64 `json:"begins"`
	Commits      int64 `json:"commits"`
	Rollbacks    int64 `json:"rollbacks"`
	Errors       int64 `json:"errors"`
	HTTPRequests int64 `json:"httpRequests"`
}

type Stat struct {
	Minute       string `json:"minute"`
	Reads        int64  `json:"reads"`
	Writes       int64  `json:"writes"`
	Begins       int64  `json:"begins"`
	Commits      int64  `json:"commits"`
	Rollbacks    int64  `json:"rollbacks"`
	Errors       int64  `json:"errors"`
	HTTPRequests int64  `json:"httpRequests"`
}

// LoadStats loads all internal stats into a LoadedStats struct.
func (db *DBStats) LoadStats() LoadedStats {
	var (
		allStats          []Stat = []Stat{}
		totalReads        int64
		totalWrites       int64
		totalBegins       int64
		totalCommits      int64
		totalRollbacks    int64
		totalErrors       int64
		totalHTTPRequests int64
	)

	db.minutes.Range(func(key, value any) bool {
		minuteKey := key.(string)
		md := value.(*minuteData)

		r := md.reads.Load()
		w := md.writes.Load()
		b := md.begins.Load()
		c := md.commits.Load()
		rb := md.rollbacks.Load()
		er := md.errors.Load()
		hr := md.httpRequests.Load()

		totalReads += r
		totalWrites += w
		totalBegins += b
		totalCommits += c
		totalRollbacks += rb
		totalErrors += er
		totalHTTPRequests += hr

		allStats = append(allStats, Stat{
			Minute:       minuteKey,
			Reads:        r,
			Writes:       w,
			Begins:       b,
			Commits:      c,
			Rollbacks:    rb,
			Errors:       er,
			HTTPRequests: hr,
		})

		return true
	})

	sort.Slice(allStats, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, allStats[i].Minute)
		tj, _ := time.Parse(time.RFC3339, allStats[j].Minute)
		return tj.Before(ti)
	})

	return LoadedStats{
		Totals: Totals{
			Reads:        totalReads,
			Writes:       totalWrites,
			Begins:       totalBegins,
			Commits:      totalCommits,
			Rollbacks:    totalRollbacks,
			Errors:       totalErrors,
			HTTPRequests: totalHTTPRequests,
		},
		Stats:              allStats,
		QueuedBegins:       db.queuedBegins.Load(),
		QueuedWrites:       db.queuedWrites.Load(),
		QueuedHTTPRequests: db.queuedHTTPRequests.Load(),
		StartedAt:          db.startedAt.Format(time.RFC3339),
		Uptime:             time.Since(db.startedAt).Round(time.Second).String(),
	}
}
