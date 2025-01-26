package nsqlitebench

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type benchmarkComplexConfig struct {
	insertXUsers              int
	insertYArticlesPerUser    int
	insertZCommentsPerArticle int
	insertGoroutines          int
}

// benchmarkComplex inserts X users, each with Y articles, and each article
// with Z comments. Then it query all users, articles, and comments with
// a JOIN query.
//
// each goroutine inserts one user with all articles and comments.
func runBenchmarkComplex(
	ctx context.Context, ciMode bool,
	db *sql.DB, fullConfig benchmarksConfig,
) (benchmarkResult, error) {
	conf := fullConfig.benchmarkComplexConfig
	start := time.Now()
	var totalReads, totalWrites uint64

	wgU := sync.WaitGroup{}
	chU := make(chan bool, conf.insertGoroutines)
	errU := make(chan error, conf.insertXUsers)
	bar := NewBar(ciMode, fmt.Sprintf("Inserting %d users", conf.insertXUsers), conf.insertXUsers)

	for idx := range conf.insertXUsers {
		wgU.Add(1)
		chU <- true
		go func() {
			defer func() {
				wgU.Done()
				<-chU
			}()
			res, err := db.ExecContext(
				ctx,
				"INSERT INTO users (created, email, active) VALUES (?, ?, ?)",
				time.Now().Unix(), fmt.Sprintf("user%d@example.com", idx), 1,
			)
			if err != nil {
				errU <- err
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				errU <- err
				return
			}

			bar.Inc()
			atomic.AddUint64(&totalWrites, uint64(affected))
		}()
	}

	wgU.Wait()
	close(chU)
	close(errU)

	for e := range errU {
		if e != nil {
			return benchmarkResult{}, fmt.Errorf("error inserting users: %w", e)
		}
	}
	bar.Finish()

	totalArticles := conf.insertXUsers * conf.insertYArticlesPerUser
	wgA := sync.WaitGroup{}
	chA := make(chan bool, conf.insertGoroutines)
	errA := make(chan error, totalArticles)
	bar = NewBar(ciMode, fmt.Sprintf("Inserting %d articles", totalArticles), totalArticles)

	for idx := range totalArticles {
		wgA.Add(1)
		chA <- true
		go func() {
			defer func() {
				wgA.Done()
				<-chA
			}()
			userID := (idx % conf.insertXUsers) + 1
			res, err := db.ExecContext(
				ctx,
				"INSERT INTO articles (created, userId, text) VALUES (?, ?, ?)",
				time.Now().Unix(), userID, fmt.Sprintf("article for user %d", userID),
			)
			if err != nil {
				errA <- err
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				errA <- err
				return
			}

			bar.Inc()
			atomic.AddUint64(&totalWrites, uint64(affected))
		}()
	}

	wgA.Wait()
	close(chA)
	close(errA)

	for e := range errA {
		if e != nil {
			return benchmarkResult{}, fmt.Errorf("error inserting articles: %w", e)
		}
	}
	bar.Finish()

	totalComments := totalArticles * conf.insertZCommentsPerArticle
	wgC := sync.WaitGroup{}
	chC := make(chan bool, conf.insertGoroutines)
	errC := make(chan error, totalComments)
	bar = NewBar(ciMode, fmt.Sprintf("Inserting %d comments", totalComments), totalComments)

	for idx := range totalComments {
		wgC.Add(1)
		chC <- true
		go func() {
			defer func() {
				wgC.Done()
				<-chC
			}()
			articleID := (idx % totalArticles) + 1
			res, err := db.ExecContext(
				ctx,
				"INSERT INTO comments (created, articleId, text) VALUES (?, ?, ?)",
				time.Now().Unix(), articleID, "comment",
			)
			if err != nil {
				errC <- err
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				errC <- err
				return
			}

			bar.Inc()
			atomic.AddUint64(&totalWrites, uint64(affected))
		}()
	}

	wgC.Wait()
	close(chC)
	close(errC)

	for e := range errC {
		if e != nil {
			return benchmarkResult{}, fmt.Errorf("error inserting comments: %w", e)
		}
	}
	bar.Finish()

	bar = NewBar(ciMode, "Reading users, articles, and comments", 1)
	rows, err := db.QueryContext(
		ctx,
		`
			SELECT
			users.id, users.created, users.email, users.active,
			articles.id, articles.created, articles.userId, articles.text,
			comments.id, comments.created, comments.articleId, comments.text
			FROM users
			LEFT JOIN articles ON articles.userId = users.id
			LEFT JOIN comments ON comments.articleId = articles.id
			ORDER BY users.created, articles.created, comments.created
		`)
	if err != nil {
		return benchmarkResult{}, fmt.Errorf("error querying: %w", err)
	}

	for rows.Next() {
		var userId, created, active int
		var email string
		var articleId, articleCreated, articleUserId int
		var articleText string
		var commentId, commentCreated, commentArticleId int
		var commentText string

		err = rows.Scan(
			&userId, &created, &email, &active,
			&articleId, &articleCreated, &articleUserId, &articleText,
			&commentId, &commentCreated, &commentArticleId, &commentText,
		)
		if err != nil {
			return benchmarkResult{}, fmt.Errorf("error when scanning: %w", err)
		}

		atomic.AddUint64(&totalReads, 1)
	}

	bar.Finish()
	return benchmarkResult{
		Name:        "Complex",
		Duration:    time.Since(start),
		TotalReads:  totalReads,
		TotalWrites: totalWrites,
	}, nil
}
