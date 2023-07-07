package middleware

import (
	"net/http"
	"time"
)

func Timeout(timeout time.Duration) Middleware {
	return func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, timeout, "Request timed out")
	}
}
