package nsqlitebench

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nsqlite/nsqlite/internal/nsqlite/styled"
	"github.com/nsqlite/nsqlite/internal/util/httputil"
	"github.com/nsqlite/nsqlite/internal/util/netutil"
	"github.com/nsqlite/nsqlite/internal/version"
)

// benchmarkResult stores the outcome of a benchmark.
type benchmarkResult struct {
	Name        string
	Duration    time.Duration
	TotalReads  uint64
	TotalWrites uint64
}

func Run(ctx context.Context, stop context.CancelFunc) error {
	fmt.Println(version.BenchVersion())
	fmt.Println()
	config := promptConfig()

	tmpDir, err := os.MkdirTemp("", "nsqlitebench_*")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	nsqliteDSN, err := startNsqlited(ctx, tmpDir)
	if err != nil {
		return fmt.Errorf("error starting nsqlited server: %w", err)
	}

	nsqliteDb, err := createNsqliteDriver(nsqliteDSN)
	if err != nil {
		return fmt.Errorf("error opening nsqlite/nsqlitego db: %w", err)
	}
	defer nsqliteDb.Close()

	mattnDb, err := createMattnDriver(tmpDir)
	if err != nil {
		return fmt.Errorf("error opening mattn/go-sqlite3 db: %w", err)
	}
	defer mattnDb.Close()

	drivers := []struct {
		Name string
		DB   *sql.DB
	}{
		{Name: "mattn/go-sqlite3", DB: mattnDb},
		{Name: "nsqlite/nsqlitego", DB: nsqliteDb},
	}

	fmt.Print("Starting in ")
	for i := 3; i > 0; i-- {
		fmt.Printf("%d..", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println("Go!")

	results := []struct {
		Name   string
		Result []benchmarkResult
	}{}

	for _, driver := range drivers {
		result, err := runBenchmark(ctx, driver.Name, driver.DB, config)
		if err != nil {
			return fmt.Errorf("error benchmarking %s: %w", driver.Name, err)
		}
		results = append(results, struct {
			Name   string
			Result []benchmarkResult
		}{
			Name:   driver.Name,
			Result: result,
		})
	}

	for _, result := range results {
		fmt.Printf("\n--- Benchmarks for %s ---\n", result.Name)
		printResults(result.Result)
	}

	<-ctx.Done()
	return nil
}

// startNsqlited starts the nsqlited server in a background goroutine.
func startNsqlited(ctx context.Context, tmpDir string) (string, error) {
	nsqlitePort, err := netutil.GetFreePort()
	if err != nil {
		return "", fmt.Errorf("error getting free port: %w", err)
	}
	dsn := fmt.Sprintf("http://localhost:%d", nsqlitePort)

	nsqliteDBDir := path.Join(tmpDir, "/nsqlite")
	if err := os.MkdirAll(nsqliteDBDir, 0755); err != nil {
		return "", fmt.Errorf("error creating temporary NSQLite database directory: %w", err)
	}

	errCh := make(chan error, 1)

	go func() {
		cmd := exec.CommandContext(
			ctx,
			"nsqlited",
			"--listen-port", strconv.Itoa(nsqlitePort),
			"--data-dir", nsqliteDBDir,
		)
		cmd.Stderr = os.Stderr

		err := cmd.Start()
		if err != nil {
			errCh <- fmt.Errorf("Error running nsqlited server: %w", err)
		}

		errCh <- nil
		_ = cmd.Wait()
	}()

	if err := <-errCh; err != nil {
		return "", err
	}

	err = httputil.WaitForServer(fmt.Sprintf("%s/health", dsn), 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("error waiting for nsqlited server: %w", err)
	}

	fmt.Printf("Temporary NSQLite database directory: %s\n", nsqliteDBDir)
	return dsn, nil
}

func printResults(results []benchmarkResult) {
	tw := styled.NewTableWriter()
	tw.AppendHeader(table.Row{"Name", "Reads", "Writes", "Duration"})

	for _, r := range results {
		tw.AppendRow(table.Row{r.Name, r.TotalReads, r.TotalWrites, r.Duration})
	}

	fmt.Println(tw.Render())
}

// runBenchmark executes all benchmarks, and returns results.
//
// It recreates the schema before each benchmark.
func runBenchmark(ctx context.Context, name string, db *sql.DB, cfg benchmarksConfig) ([]benchmarkResult, error) {
	fmt.Printf("\n--- Benchmarking %s ---\n", name)

	if err := recreateSchema(ctx, db); err != nil {
		return nil, err
	}

	benchs := []func(context.Context, *sql.DB, benchmarksConfig) (benchmarkResult, error){
		runBenchmarkSimple,
		runBenchmarkComplex,
		runBenchmarkMany,
		runBenchmarkLarge,
	}

	var results []benchmarkResult

	for _, bench := range benchs {
		if err := recreateSchema(ctx, db); err != nil {
			return nil, err
		}

		res, err := bench(ctx, db, cfg)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	return results, nil
}
