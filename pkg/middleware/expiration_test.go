package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/user"
	"github.com/stretchr/testify/assert"
)

func TestExpiration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       *user.Env
		code        int
		body        string
	}{
		{
			"instance has not expired",
			&user.Env{Expiration: time.Now().AddDate(0, 0, 1)},
			200,
			``,
		},
		{
			"instance has expired",
			&user.Env{Expiration: time.Now().AddDate(0, 0, -1)},
			503,
			`The service instance has expired`,
		},
		{
			"invalid instance without expiration date",
			&user.Env{},
			503,
			`The service instance has expired`,
		},
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var body bytes.Buffer

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(io.Discard).Sugar()

			expected := &gabi.Config{Logger: logger, UserEnv: tc.given}
			Expiration(expected)(dummyHandler).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, body.String(), tc.body)
		})
	}
}
