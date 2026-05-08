package stats

import (
	"sync"
	"sync/atomic"
	"time"
)

// minuteData holds the counters for a specific minute (atomic for thread safety).
type minuteData struct {
	reads        atomic.Int64
	writes       atomic.Int64
	begins       atomic.Int64
	commits      atomic.Int64
	rollbacks    atomic.Int64
	errors       atomic.Int64
	httpRequests atomic.Int64
}

// DBStats holds the stats for the database.
type DBStats struct {
	startedAt          time.Time
	minutes            sync.Map // key: string (minute RFC3339) -> value: *minuteData
	queuedBegins       atomic.Int64
	queuedWrites       atomic.Int64
	queuedHTTPRequests atomic.Int64
	stopChan           chan bool
}

// NewDBStats creates a DBStats instance.
func NewDBStats() *DBStats {
	db := &DBStats{
		startedAt: time.Now().UTC(),
		stopChan:  make(chan bool),
	}
	go db.runCleanupWorker()
	return db
}

// Close stops the background cleanup worker.
func (db *DBStats) Close() {
	close(db.stopChan)
}

// runCleanupWorker removes stats older than 24 hours every 10 seconds.
func (db *DBStats) runCleanupWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-24 * time.Hour)
			db.minutes.Range(func(key, value any) bool {
				minuteStr := key.(string)
				t, err := time.Parse(time.RFC3339, minuteStr)
				if err != nil {
					return true
				}
				if t.Before(cutoff) {
					db.minutes.Delete(key)
				}
				return true
			})
		case <-db.stopChan:
			return
		}
	}
}

// getOrCreateMinuteData returns a *minuteData for the current minute (UTC).
// If none exists, it creates one.
func (db *DBStats) getOrCreateMinuteData() *minuteData {
	minuteKey := time.Now().UTC().Truncate(time.Minute).Format(time.RFC3339)
	val, ok := db.minutes.Load(minuteKey)
	if !ok {
		md := &minuteData{}
		actual, loaded := db.minutes.LoadOrStore(minuteKey, md)
		if loaded {
			return actual.(*minuteData)
		}
		return md
	}
	return val.(*minuteData)
}

// IncReads increments the read counter for the current minute.
func (db *DBStats) IncReads() {
	md := db.getOrCreateMinuteData()
	md.reads.Add(1)
}

// IncWrites increments the write counter for the current minute.
func (db *DBStats) IncWrites() {
	md := db.getOrCreateMinuteData()
	md.writes.Add(1)
}

// IncBegins increments the begin counter for the current minute.
func (db *DBStats) IncBegins() {
	md := db.getOrCreateMinuteData()
	md.begins.Add(1)
}

// IncCommits increments the commit counter for the current minute.
func (db *DBStats) IncCommits() {
	md := db.getOrCreateMinuteData()
	md.commits.Add(1)
}

// IncRollbacks increments the rollback counter for the current minute.
func (db *DBStats) IncRollbacks() {
	md := db.getOrCreateMinuteData()
	md.rollbacks.Add(1)
}

// IncErrors increments the error counter for the current minute.
func (db *DBStats) IncErrors() {
	md := db.getOrCreateMinuteData()
	md.errors.Add(1)
}

// IncHTTPRequests increments the HTTP requests counter for the current minute.
func (db *DBStats) IncHTTPRequests() {
	md := db.getOrCreateMinuteData()
	md.httpRequests.Add(1)
}

// IncQueuedBegins increments the queued begins counter atomically.
func (db *DBStats) IncQueuedBegins() {
	db.queuedBegins.Add(1)
}

// DecQueuedBegins decrements the queued begins counter atomically.
func (db *DBStats) DecQueuedBegins() {
	db.queuedBegins.Add(-1)
}

// IncQueuedWrites increments the queued writes counter atomically.
func (db *DBStats) IncQueuedWrites() {
	db.queuedWrites.Add(1)
}

// DecQueuedWrites decrements the queued writes counter atomically.
func (db *DBStats) DecQueuedWrites() {
	db.queuedWrites.Add(-1)
}

// IncQueuedHTTPRequests increments the queued HTTP requests counter atomically.
func (db *DBStats) IncQueuedHTTPRequests() {
	db.queuedHTTPRequests.Add(1)
}

// DecQueuedHTTPRequests decrements the queued HTTP requests counter atomically.
func (db *DBStats) DecQueuedHTTPRequests() {
	db.queuedHTTPRequests.Add(-1)
}
