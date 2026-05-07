package query_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

const numConcurrentWorkers = 500

func TestConcurrentTransactionsPreserveIntegrityWithBatchAndSeparateRequests(t *testing.T) {
	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{
			Query: "CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, worker_id INTEGER NOT NULL, tag TEXT NOT NULL);",
		},
	)

	baseline := server.Stats(t, "")
	client := &http.Client{}

	errs := make([]error, 0, numConcurrentWorkers)
	var errsMu sync.Mutex
	var completed atomic.Int64
	var wg sync.WaitGroup

	for workerOffset := range numConcurrentWorkers {
		workerID := workerOffset + 1
		batchMode := workerID%2 == 0
		wg.Go(func() {
			if err := runWorkerTransaction(
				client,
				server.BaseURL(),
				workerID,
				batchMode,
			); err != nil {
				errsMu.Lock()
				errs = append(errs, fmt.Errorf("worker %d: %w", workerID, err))
				errsMu.Unlock()
				return
			}
			completed.Add(1)
		})
	}

	wg.Wait()
	require.Empty(t, errs, "%v", errs)
	require.Equal(t, int64(numConcurrentWorkers), completed.Load())

	result := server.Query(t, "", harness.Query{
		Query: "SELECT COUNT(*) FROM items;",
	})
	require.Equal(t, [][]any{{float64(numConcurrentWorkers)}}, result.Results[0].Rows)

	final := server.Stats(t, "")

	require.Equal(t, baseline.Totals.Begins+numConcurrentWorkers, final.Totals.Begins)
	require.Equal(t, baseline.Totals.Commits+numConcurrentWorkers, final.Totals.Commits)

	require.Equal(t, baseline.Totals.Errors, final.Totals.Errors)
	require.Equal(t, baseline.Totals.Rollbacks, final.Totals.Rollbacks)
	require.Zero(t, final.QueuedBegins)
	require.Zero(t, final.QueuedWrites)
	require.Zero(t, final.QueuedHTTPRequests)
}

func runWorkerTransaction(
	client *http.Client,
	baseURL string,
	workerID int,
	batchMode bool,
) error {
	beginResp, err := sendQuery(client, baseURL, harness.Query{Query: "BEGIN;"})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if len(beginResp.Results) != 1 || beginResp.Results[0].Type != "begin" {
		return fmt.Errorf("unexpected begin response: %+v", beginResp.Results)
	}
	txID := beginResp.Results[0].TxID
	if txID == "" {
		return fmt.Errorf("empty txId")
	}

	body := []harness.Query{
		{
			TxID:   txID,
			Query:  "INSERT INTO items (worker_id, tag) VALUES (?, 'first');",
			Params: []harness.QueryParam{{Value: workerID}},
		},
		{
			TxID:   txID,
			Query:  "INSERT INTO items (worker_id, tag) VALUES (?, 'second');",
			Params: []harness.QueryParam{{Value: workerID}},
		},
		{
			TxID:   txID,
			Query:  "SELECT COUNT(*) FROM items WHERE worker_id = ?;",
			Params: []harness.QueryParam{{Value: workerID}},
		},
		{
			TxID:   txID,
			Query:  "DELETE FROM items WHERE worker_id = ? AND tag = 'first';",
			Params: []harness.QueryParam{{Value: workerID}},
		},
		{TxID: txID, Query: "COMMIT;"},
	}

	if batchMode {
		return validateBatchTransaction(client, baseURL, body)
	}
	return validateSeparateTransaction(client, baseURL, body)
}

func validateBatchTransaction(
	client *http.Client,
	baseURL string,
	queries []harness.Query,
) error {
	resp, err := sendQuery(client, baseURL, queries...)
	if err != nil {
		return fmt.Errorf("batch: %w", err)
	}
	if len(resp.Results) != 5 {
		return fmt.Errorf("expected 5 results in batch, got %d", len(resp.Results))
	}
	if resp.Results[0].Type != "write" {
		return fmt.Errorf("batch result[0] expected write, got %s", resp.Results[0].Type)
	}
	if resp.Results[1].Type != "write" {
		return fmt.Errorf("batch result[1] expected write, got %s", resp.Results[1].Type)
	}
	if resp.Results[2].Type != "read" {
		return fmt.Errorf("batch result[2] expected read, got %s", resp.Results[2].Type)
	}
	if resp.Results[3].Type != "write" {
		return fmt.Errorf("batch result[3] expected write (delete), got %s", resp.Results[3].Type)
	}
	if resp.Results[4].Type != "commit" {
		return fmt.Errorf("batch result[4] expected commit, got %s", resp.Results[4].Type)
	}
	return nil
}

func validateSeparateTransaction(
	client *http.Client,
	baseURL string,
	queries []harness.Query,
) error {
	for i, query := range queries {
		resp, err := sendQuery(client, baseURL, query)
		if err != nil {
			return fmt.Errorf("query %d: %w", i, err)
		}
		if len(resp.Results) != 1 {
			return fmt.Errorf("query %d expected 1 result, got %d", i, len(resp.Results))
		}
	}
	return nil
}

func sendQuery(
	client *http.Client,
	baseURL string,
	queries ...harness.Query,
) (harness.QueryResponse, error) {
	body, err := json.Marshal(queries)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		baseURL+"/query",
		bytes.NewReader(body),
	)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return harness.QueryResponse{}, fmt.Errorf(
			"unexpected status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	var qr harness.QueryResponse
	if err := json.Unmarshal(respBody, &qr); err != nil {
		return harness.QueryResponse{}, fmt.Errorf("decode response body: %w", err)
	}

	return qr.WithoutTiming(), nil
}
