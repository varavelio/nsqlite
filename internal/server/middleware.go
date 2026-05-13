package server

import "net/http"

// maxRequestSize returns the maximum HTTP request body size in bytes.
func (s *Server) maxRequestSize() int64 {
	if s.MaxRequestSizeMB <= 0 {
		return 100 * 1024 * 1024 // 100 MB default
	}
	return int64(s.MaxRequestSizeMB) * 1024 * 1024
}

// maxRequestBodyMiddleware wraps the next handler with a request body size limit.
func (s *Server) maxRequestBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize())
		next.ServeHTTP(w, r)
	})
}
