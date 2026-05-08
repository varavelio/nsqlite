package server

import "net/http"

func (s *Server) maxRequestSize() int64 {
	if s.MaxRequestSizeMB <= 0 {
		return 100 * 1024 * 1024
	}
	return int64(s.MaxRequestSizeMB) * 1024 * 1024
}

func (s *Server) maxRequestBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize())
		next.ServeHTTP(w, r)
	})
}
