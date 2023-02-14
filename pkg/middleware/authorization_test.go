package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/user"
	"github.com/stretchr/testify/assert"
)

func TestAuthorization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       *user.Env
		headers     func(*http.Request)
		code        int
		body        string
		user        string
	}{
		{
			"no users set with valid header",
			&user.Env{},
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-User", "test")
			},
			401,
			`Request cannot be authorized`,
			``,
		},
		{
			"empty users lists with valid header",
			&user.Env{Users: []string{}},
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-User", "test")
			},
			401,
			`Request cannot be authorized`,
			``,
		},
		{
			"empty users lists with invalid header",
			&user.Env{Users: []string{}},
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "test")
			},
			400,
			`Request without required header: X-Forwarded-User`,
			``,
		},
		{
			"no users set without required header",
			&user.Env{},
			func(r *http.Request) {
				// No-op.
			},
			400,
			`Request without required header: X-Forwarded-User`,
			``,
		},
		{
			"users set without required header",
			&user.Env{Users: []string{"test"}},
			func(r *http.Request) {
				// No-op.
			},
			400,
			`Request without required header: X-Forwarded-User`,
			``,
		},
		{
			"users set with required header value set to invalid user",
			&user.Env{Users: []string{"test"}},
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-User", "test2")
			},
			403,
			`User does not have required permissions`,
			``,
		},
		{
			"users set with required header value set to valid user",
			&user.Env{Users: []string{"test"}},
			func(r *http.Request) {
				r.Header.Set("X-Forwarded-User", "test")
			},
			200,
			``,
			`test`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var (
				body bytes.Buffer
				user string
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(io.Discard).Sugar()

			tc.headers(r)

			expected := &gabi.Config{Logger: logger, UserEnv: tc.given}
			Authorization(expected)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s, ok := r.Context().Value(ContextKeyUser).(string)
				if !ok {
					t.Fatal("invalid context")
				}
				user = s
			})).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, body.String(), tc.body)
			assert.Equal(t, tc.user, user)
		})
	}
}
