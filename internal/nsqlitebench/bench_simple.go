package nsqlitebench

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type benchmarkSimpleConfig struct {
	insertXUsers     int
	queryYUsers      int
	insertGoroutines int
	queryGoroutines  int
}

// runBenchmarkSimple inserts X users and then queries all of them in single
// query.
//
// This also reads the users Y times.
func runBenchmarkSimple(
	ctx context.Context, db *sql.DB, fullConfig benchmarksConfig,
) (benchmarkResult, error) {
	conf := fullConfig.benchmarkSimpleConfig
	start := time.Now()
	var totalReads uint64 = 0
	var totalWrites uint64 = 0

	wg := sync.WaitGroup{}
	wgch := make(chan bool, conf.insertGoroutines)
	bar := NewBar(fmt.Sprintf("Inserting %d users", conf.insertXUsers), conf.insertXUsers)

	for idx := range conf.insertXUsers {
		wg.Add(1)
		wgch <- true

		go func() {
			defer func() {
				wg.Done()
				<-wgch
			}()

			res, err := db.ExecContext(
				ctx,
				"INSERT INTO users (created, email, active) VALUES (?, ?, ?)",
				time.Now().Unix(), fmt.Sprintf("user%d@example.com", idx), 1,
			)
			if err != nil {
				panic(err)
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				panic(err)
			}

			bar.Inc()
			atomic.AddUint64(&totalWrites, uint64(rowsAffected))
		}()
	}

	wg.Wait()
	close(wgch)

	bar.Finish()
	bar = NewBar("Reading all users in single query", 1)

	rows, err := db.QueryContext(
		ctx,
		"SELECT id, created, email, active FROM users ORDER BY id",
	)
	if err != nil {
		return benchmarkResult{}, fmt.Errorf("error when querying: %w", err)
	}

	for rows.Next() {
		var id, created, active int
		var email string
		err = rows.Scan(&id, &created, &email, &active)
		if err != nil {
			return benchmarkResult{}, fmt.Errorf("error when scanning: %w", err)
		}
		atomic.AddUint64(&totalReads, 1)
	}
	bar.Finish()

	bar = NewBar(fmt.Sprintf("Reading users %d times", conf.queryYUsers), conf.queryYUsers)
	wg = sync.WaitGroup{}
	wgch = make(chan bool, conf.queryGoroutines)

	for idx := range conf.queryYUsers {
		wg.Add(1)
		wgch <- true
		userID := max(idx%conf.insertXUsers, 1)

		go func() {
			defer func() {
				wg.Done()
				<-wgch
			}()

			rows, err := db.QueryContext(
				ctx,
				"SELECT id, created, email, active FROM users WHERE id = ?",
				userID,
			)
			if err != nil {
				panic(err)
			}

			for rows.Next() {
				var id, created, active int
				var email string
				err = rows.Scan(&id, &created, &email, &active)
				if err != nil {
					panic(err)
				}
			}

			bar.Inc()
			atomic.AddUint64(&totalReads, 1)
		}()
	}

	wg.Wait()
	close(wgch)

	bar.Finish()
	return benchmarkResult{
		Name:        "Simple",
		Duration:    time.Since(start),
		TotalReads:  totalReads,
		TotalWrites: totalWrites,
	}, nil
}
