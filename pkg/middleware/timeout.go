package middleware

import (
	"context"
	"net/http"
	"time"
)

func Timeout(timeout time.Duration) Middleware {
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, timeout, "Request timed out")
	}
}

// ContextTimeout sets a context deadline without wrapping the ResponseWriter.
// Unlike http.TimeoutHandler this does not buffer writes, so the handler can
// stream directly to the client. The DB driver and any context-aware I/O will
// still respect the deadline; the handler is responsible for checking ctx.Err().
func ContextTimeout(timeout time.Duration) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
