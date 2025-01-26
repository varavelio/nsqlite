package nsqlitebench

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type benchmarkManyConfig struct {
	insertXUsers     int
	queryUsersYTimes int
	insertGoroutines int
	queryGoroutines  int
}

// runBenchmarkMany inserts X users in a single transaction and then query all
// users Y times. This simulates a read-heavy workload.
func runBenchmarkMany(
	ctx context.Context, db *sql.DB, fullConfig benchmarksConfig,
) (benchmarkResult, error) {
	conf := fullConfig.benchmarkManyConfig
	start := time.Now()
	var totalReads, totalWrites uint64

	tx, err := db.Begin()
	if err != nil {
		return benchmarkResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(
		ctx,
		"INSERT INTO users (created, email, active) VALUES (?, ?, ?)",
	)
	if err != nil {
		return benchmarkResult{}, err
	}
	defer func() { _ = stmt.Close() }()

	wgInsert := sync.WaitGroup{}
	chInsert := make(chan bool, conf.insertGoroutines)
	errInsert := make(chan error, conf.insertXUsers)
	bar := NewBar(
		fmt.Sprintf("Inserting %d users", conf.insertXUsers), conf.insertXUsers,
	)

	for idx := range conf.insertXUsers {
		wgInsert.Add(1)
		chInsert <- true
		go func() {
			defer func() {
				wgInsert.Done()
				<-chInsert
			}()
			res, err := stmt.ExecContext(
				ctx,
				time.Now().Unix(), fmt.Sprintf("user%d@example.com", idx), 1,
			)
			if err != nil {
				errInsert <- err
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				errInsert <- err
				return
			}

			bar.Inc()
			atomic.AddUint64(&totalWrites, uint64(affected))
		}()
	}

	wgInsert.Wait()
	close(chInsert)
	close(errInsert)

	for e := range errInsert {
		if e != nil {
			return benchmarkResult{}, e
		}
	}
	if err := tx.Commit(); err != nil {
		return benchmarkResult{}, err
	}
	bar.Finish()

	wgQuery := sync.WaitGroup{}
	chQuery := make(chan bool, conf.queryGoroutines)
	errQuery := make(chan error, conf.queryUsersYTimes)
	bar = NewBar(
		fmt.Sprintf("Querying all users %d times", conf.queryUsersYTimes),
		conf.queryUsersYTimes,
	)

	for i := 0; i < conf.queryUsersYTimes; i++ {
		wgQuery.Add(1)
		chQuery <- true
		go func() {
			defer func() {
				wgQuery.Done()
				<-chQuery
			}()
			rows, err := db.QueryContext(
				ctx,
				"SELECT id, created, email, active FROM users ORDER BY id",
			)
			if err != nil {
				errQuery <- err
				return
			}
			defer rows.Close()

			for rows.Next() {
				var id, created, active int
				var email string
				if err := rows.Scan(&id, &created, &email, &active); err != nil {
					errQuery <- err
					return
				}
				atomic.AddUint64(&totalReads, 1)
			}

			bar.Inc()
		}()
	}

	wgQuery.Wait()
	close(chQuery)
	close(errQuery)

	for e := range errQuery {
		if e != nil {
			return benchmarkResult{}, e
		}
	}
	bar.Finish()

	return benchmarkResult{
		Name:        "Many",
		Duration:    time.Since(start),
		TotalReads:  totalReads,
		TotalWrites: totalWrites,
	}, nil
}
