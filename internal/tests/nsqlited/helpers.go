package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/nsqlite/nsqlite/internal/nsqlited"
	"github.com/nsqlite/nsqlite/internal/nsqlited/config"
	"github.com/nsqlite/nsqlite/internal/nsqlited/server"
	"github.com/nsqlite/nsqlite/internal/util/httputil"
	"github.com/nsqlite/nsqlite/internal/util/netutil"
	"github.com/stretchr/testify/assert"
)

// createServer creates a new NSQLite server and returns the
// http base url to use in tests.
//
// The purpose of this function is to create a new NSQLite server for each
// test and the data is cleaned up automatically when the test finishes.
func createServer(t testing.TB, conf ...config.Config) string {
	t.Helper()

	port, err := netutil.GetFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "nsqlite_integration_test_*")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	pickedConf := config.Config{}
	if len(conf) > 0 {
		pickedConf = conf[0]
	}
	pickedConf.DataDir = tmpDir
	pickedConf.ListenPort = strconv.Itoa(port)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel() })

	go func() {
		_ = nsqlited.Run(ctx, cancel, pickedConf.ToArgs())
	}()

	baseURL := fmt.Sprintf("http://localhost:%s", pickedConf.ListenPort)
	if err := httputil.WaitForServer(baseURL+"/health", 5*time.Second); err != nil {
		t.Fatalf("failed to wait for the server to start: %v", err)
	}

	return baseURL
}

// sendQuery sends a query to a NSQLite server and returns the response
// seting all the time fields to 0 to make the results deterministic.
//
// This function asserts that the response is successful, should be used
// to test successful queries.
func sendQuery(t testing.TB, url string, query server.Query) server.Response {
	t.Helper()

	reqBody, err := json.Marshal([]server.Query{query})
	assert.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	var resBody server.Response
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&resBody))

	resBody.Time = 0
	for i := range resBody.Results {
		resBody.Results[i].Time = 0
	}

	return resBody
}
