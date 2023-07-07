package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       http.HandlerFunc
		expected    time.Duration
		context     func() context.Context
		code        int
		body        string
	}{
		{
			"no timeout with HTTP request without delay",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// No-op.
			}),
			time.Duration(10 * time.Millisecond),
			func() context.Context {
				return context.TODO()
			},
			200,
			``,
		},
		{
			"no timeout with HTTP request with delay",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Millisecond)
			}),
			time.Duration(10 * time.Millisecond),
			func() context.Context {
				return context.TODO()
			},
			200,
			``,
		},
		{
			"timeout with HTTP request with delay exceeding limit",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Millisecond)
			}),
			time.Duration(1 * time.Millisecond),
			func() context.Context {
				return context.Background()
			},
			503,
			`Request timed out`,
		},
		{
			"no timeout with HTTP request error due to cancelled context",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// No-op.
			}),
			time.Duration(1 * time.Millisecond),
			func() context.Context {
				ctx, cancel := context.WithCancel(context.TODO())
				cancel()
				return ctx
			},
			503,
			``,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var body bytes.Buffer

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			Timeout(tc.expected)(tc.given).ServeHTTP(w, r.WithContext(tc.context()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, body.String(), tc.body)
		})
	}
}
