package runtime_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/nsqlite/e2e/harness"
)

func TestBinaryDoesNotServeUIFromRoot(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{})
	response := server.Get(t, "/", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestBinaryHandlesConfiguredCORSPreflight(t *testing.T) {
	t.Parallel()

	server := harness.StartServer(t, harness.ServerConfig{
		CORSAllowedOrigins:   "https://console.example.com",
		CORSAllowCredentials: true,
	})

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodOptions,
		server.BaseURL()+harness.DatabaseQueryPath,
		nil,
	)
	require.NoError(t, err)
	request.Header.Set("Origin", "https://console.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.Equal(
		t,
		"https://console.example.com",
		response.Header.Get("Access-Control-Allow-Origin"),
	)
	require.Equal(t, "true", response.Header.Get("Access-Control-Allow-Credentials"))
	require.Equal(
		t,
		"GET, HEAD, POST, OPTIONS",
		response.Header.Get("Access-Control-Allow-Methods"),
	)
	require.Equal(
		t,
		"Accept, Authorization, Content-Type",
		response.Header.Get("Access-Control-Allow-Headers"),
	)
}
