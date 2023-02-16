package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       http.HandlerFunc
		code        int
		error       bool
		want        string
	}{
		{
			"no panic",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// No-op.
			}),
			200,
			false,
			``,
		},
		{
			"panic with recovery",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test")
			}),
			500,
			false,
			`Recovered from an error: test`,
		},
		{
			"panic without recovery with special error type",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(http.ErrAbortHandler)
			}),
			0,
			true,
			``,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			defer func() { _ = recover() }()

			var body, output bytes.Buffer

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(&output).Sugar()

			expected := &gabi.Config{Logger: logger}
			Recovery(expected)(tc.given).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, output.String(), tc.want)

			if tc.error {
				t.Fatal("did not panic")
			}
		})
	}
}
