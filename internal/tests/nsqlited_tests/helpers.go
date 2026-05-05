package nsqlited_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/nsqlite/nsqlite/internal/nsqlited"
	"github.com/nsqlite/nsqlite/internal/nsqlited/config"
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
		_ = nsqlited.Run(ctx, cancel, io.Discard, pickedConf.ToArgs())
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
func sendQuery(t testing.TB, url string, query Query) Response {
	t.Helper()

	reqBody, err := json.Marshal([]Query{query})
	assert.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	var resBody Response
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&resBody))

	resBody.Time = 0
	for i := range resBody.Results {
		resBody.Results[i].Time = 0
	}

	return resBody
}

// assertQuery sends a query to a NSQLite server and asserts that the response
// is successful and equals the expected response.
//
// This function converts all the time fields to 0 to make the results deterministic.
func assertQuery(t testing.TB, url string, query Query, expected Response) {
	t.Helper()

	response := sendQuery(t, url, query)
	assert.Equal(t, response, expected)
}

// assertQueryStatus sends a query to a NSQLite server and asserts
// that the response status code is the expected one.
//
// If a token is provided, it will be sent as the Authorization header, otherwise
// the request will be sent without the header.
func assertQueryStatus(t testing.TB, url, token string, query Query, expectedStatus int) {
	t.Helper()

	reqBody, err := json.Marshal([]Query{query})
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	assert.NoError(t, err)

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	assert.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, res.StatusCode, expectedStatus)
}

// getStats sends a GET request to the /stats endpoint and returns the response
//
// It sets the StartedAt and Uptime to the zero values to make the results deterministic.
func getStats(t testing.TB, baseURL string) LoadedStats {
	t.Helper()
	url := baseURL + "/stats"

	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	var resBody LoadedStats
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&resBody))

	resBody.StartedAt = ""
	resBody.Uptime = ""

	return resBody
}
