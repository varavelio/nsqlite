package httputil

import (
	"errors"
	"io"
	"net/http"
)

// ReadReqBodyBytes reads the request body from the given request and returns
// it as a byte slice.
//
// Once the body is read, it is closed and cannot be read again.
func ReadReqBodyBytes(r *http.Request) ([]byte, error) {
	if r == nil {
		return nil, errors.New("request cannot be nil")
	}

	if r.Body == nil {
		return nil, nil
	}

	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}
