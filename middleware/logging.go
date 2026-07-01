package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		attrs := []any{"method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", time.Since(start)}
		if rw.status >= 400 {
			slog.Default().Warn("http request", attrs...)
		} else {
			slog.Default().Info("http request", attrs...)
		}
	})
}
