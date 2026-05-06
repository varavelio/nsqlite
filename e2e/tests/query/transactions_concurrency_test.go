package query_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

const (
	queuedTransactionWorkerCount = 120
	queuedTransactionQueueFloor  = 30
)

func TestTransactionsQueueConcurrentBeginsAndPreserveIntegrityUnderLoad(t *testing.T) {
	server := harness.StartServer(t, harness.ServerConfig{})
	server.Query(
		t,
		"",
		harness.Query{
			Query: "CREATE TABLE account_totals (id INTEGER PRIMARY KEY, total INTEGER NOT NULL, tx_count INTEGER NOT NULL);",
		},
		harness.Query{
			Query: "CREATE TABLE audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, worker_id INTEGER NOT NULL, step TEXT NOT NULL, delta INTEGER NOT NULL);",
		},
		harness.Query{
			Query: "INSERT INTO account_totals (id, total, tx_count) VALUES (1, 0, 0);",
		},
	)

	baselineStats := server.Stats(t, "")
	holdingTransaction := server.Query(t, "", harness.Query{Query: "BEGIN;"})
	require.Equal(t, "begin", holdingTransaction.Results[0].Type)
	require.NotEmpty(t, holdingTransaction.Results[0].TxID)

	client := &http.Client{}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	start := make(chan struct{})
	errCh := make(chan error, queuedTransactionWorkerCount)
	var activationOrder atomic.Int64
	var completionOrder atomic.Int64
	var wg sync.WaitGroup

	for workerOffset := range queuedTransactionWorkerCount {
		workerID := workerOffset + 1
		wg.Go(func() {
			<-start
			if err := runQueuedTransaction(
				ctx,
				client,
				server.BaseURL(),
				workerID,
				&activationOrder,
				&completionOrder,
			); err != nil {
				errCh <- err
			}
		})
	}

	close(start)
	require.Eventually(t, func() bool {
		stats := server.Stats(t, "")
		return stats.QueuedBegins >= queuedTransactionQueueFloor
	}, 5*time.Second, 25*time.Millisecond)

	holdingCommit := server.Query(
		t,
		"",
		harness.Query{Query: "COMMIT;", TxID: holdingTransaction.Results[0].TxID},
	)
	require.Equal(t, "commit", holdingCommit.Results[0].Type)

	completed := make(chan struct{})
	go func() {
		wg.Wait()
		close(completed)
	}()

	select {
	case <-completed:
	case <-ctx.Done():
		t.Fatal("concurrent transaction workers did not complete before timeout")
	}

	close(errCh)
	workerErrors := make([]string, 0)
	for err := range errCh {
		workerErrors = append(workerErrors, err.Error())
	}
	require.Empty(t, workerErrors, "%v", workerErrors)

	finalState := server.Query(
		t,
		"",
		harness.Query{Query: "SELECT total, tx_count FROM account_totals WHERE id = 1;"},
		harness.Query{Query: "SELECT COUNT(*) FROM audit_log;"},
	)

	expectedTotal := float64(queuedTransactionWorkerCount * (queuedTransactionWorkerCount + 1) / 2)
	require.Equal(
		t,
		[][]any{{expectedTotal, float64(queuedTransactionWorkerCount)}},
		finalState.Results[0].Rows,
	)
	require.Equal(
		t,
		[][]any{{float64(queuedTransactionWorkerCount * 2)}},
		finalState.Results[1].Rows,
	)

	finalStats := server.Stats(t, "")
	require.Equal(
		t,
		baselineStats.Totals.Begins+queuedTransactionWorkerCount+1,
		finalStats.Totals.Begins,
	)
	require.Equal(
		t,
		baselineStats.Totals.Commits+queuedTransactionWorkerCount+1,
		finalStats.Totals.Commits,
	)
	require.Equal(t, baselineStats.Totals.Rollbacks, finalStats.Totals.Rollbacks)
	require.Equal(t, baselineStats.Totals.Errors, finalStats.Totals.Errors)
	require.Zero(t, finalStats.QueuedBegins)
	require.Zero(t, finalStats.QueuedWrites)
	require.Zero(t, finalStats.QueuedHTTPRequests)
	require.Equal(t, int64(queuedTransactionWorkerCount), activationOrder.Load())
	require.Equal(t, int64(queuedTransactionWorkerCount), completionOrder.Load())
}

func runQueuedTransaction(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	workerID int,
	activationOrder, completionOrder *atomic.Int64,
) error {
	beginResponse, err := postQueryRequest(ctx, client, baseURL, harness.Query{Query: "BEGIN;"})
	if err != nil {
		return fmt.Errorf("worker %d begin request: %w", workerID, err)
	}
	if len(beginResponse.Results) != 1 || beginResponse.Results[0].Type != "begin" {
		return fmt.Errorf(
			"worker %d unexpected begin response: %+v",
			workerID,
			beginResponse.Results,
		)
	}
	if beginResponse.Results[0].TxID == "" {
		return fmt.Errorf("worker %d received an empty txId", workerID)
	}

	txID := beginResponse.Results[0].TxID
	activationPosition := activationOrder.Add(1)

	transactionBody, err := postQueryRequest(
		ctx,
		client,
		baseURL,
		harness.Query{
			TxID:  txID,
			Query: "INSERT INTO audit_log (worker_id, step, delta) VALUES (?, ?, ?);",
			Params: []harness.QueryParam{
				{Value: workerID},
				{Value: "begin"},
				{Value: workerID},
			},
		},
		harness.Query{
			TxID:   txID,
			Query:  "UPDATE account_totals SET total = total + ?, tx_count = tx_count + 1 WHERE id = 1 RETURNING total, tx_count;",
			Params: []harness.QueryParam{{Value: workerID}},
		},
		harness.Query{
			TxID:  txID,
			Query: "INSERT INTO audit_log (worker_id, step, delta) VALUES (?, ?, ?);",
			Params: []harness.QueryParam{
				{Value: workerID},
				{Value: "before-commit"},
				{Value: workerID},
			},
		},
		harness.Query{
			TxID:  txID,
			Query: "SELECT total, tx_count FROM account_totals WHERE id = 1;",
		},
		harness.Query{TxID: txID, Query: "COMMIT;"},
	)
	if err != nil {
		return fmt.Errorf("worker %d transaction body: %w", workerID, err)
	}
	if len(transactionBody.Results) != 5 {
		return fmt.Errorf(
			"worker %d unexpected transaction result count: %d",
			workerID,
			len(transactionBody.Results),
		)
	}
	if transactionBody.Results[0].Type != "write" || transactionBody.Results[0].RowsAffected != 1 {
		return fmt.Errorf(
			"worker %d unexpected first write result: %+v",
			workerID,
			transactionBody.Results[0],
		)
	}
	if transactionBody.Results[1].Type != "write" || len(transactionBody.Results[1].Rows) != 1 {
		return fmt.Errorf(
			"worker %d unexpected update result: %+v",
			workerID,
			transactionBody.Results[1],
		)
	}
	if transactionBody.Results[2].Type != "write" || transactionBody.Results[2].RowsAffected != 1 {
		return fmt.Errorf(
			"worker %d unexpected second write result: %+v",
			workerID,
			transactionBody.Results[2],
		)
	}
	if transactionBody.Results[3].Type != "read" {
		return fmt.Errorf(
			"worker %d unexpected read result: %+v",
			workerID,
			transactionBody.Results[3],
		)
	}
	if transactionBody.Results[4].Type != "commit" {
		return fmt.Errorf(
			"worker %d unexpected commit result in batch: %+v",
			workerID,
			transactionBody.Results[4],
		)
	}
	if !reflect.DeepEqual(transactionBody.Results[1].Rows, transactionBody.Results[3].Rows) {
		return fmt.Errorf(
			"worker %d saw inconsistent in-transaction state: update=%v read=%v",
			workerID,
			transactionBody.Results[1].Rows,
			transactionBody.Results[3].Rows,
		)
	}

	completionPosition := completionOrder.Add(1)
	if activationPosition != completionPosition {
		return fmt.Errorf(
			"worker %d finished out of serialized response order: begin=%d commit=%d",
			workerID,
			activationPosition,
			completionPosition,
		)
	}

	return nil
}

func postQueryRequest(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	queries ...harness.Query,
) (harness.QueryResponse, error) {
	encodedBody, err := json.Marshal(queries)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/query",
		bytes.NewReader(encodedBody),
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return harness.QueryResponse{}, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return harness.QueryResponse{}, fmt.Errorf(
			"unexpected status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var response harness.QueryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return harness.QueryResponse{}, fmt.Errorf("decode response body: %w", err)
	}

	return response.WithoutTiming(), nil
}
