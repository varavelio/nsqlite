package httputil

import (
	"fmt"
	"net/http"
	"time"
)

// WaitForServer waits for the server at the given URL to become ready. The
// server is considered ready if it returns a 200 OK response for the given
// URL.
func WaitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("server at %s did not become ready within %s", url, timeout.String())
}
