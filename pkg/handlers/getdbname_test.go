package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentDBName(t *testing.T) {
	cases := []struct {
		description string
		dbName      string
		code        int
		body        map[string]string
	}{
		{
			"returns current database name",
			"test_db",
			200,
			map[string]string{"db_name": "test_db"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			var body bytes.Buffer

			dbEnv := &db.Env{Name: tc.dbName}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(io.Discard).Sugar()

			expected := &gabi.Config{DBEnv: dbEnv, Logger: logger}
			GetCurrentDBName(expected).ServeHTTP(w, r.WithContext(context.TODO()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			var responseBody map[string]string
			err := json.Unmarshal(body.Bytes(), &responseBody)

			require.NoError(t, err)
			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Equal(t, tc.body, responseBody)
		})
	}
}
