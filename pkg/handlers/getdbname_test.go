package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDBName(t *testing.T) {
	cases := []struct {
		description    string
		dbName         string
		defaultDBName  string
		expectedStatus int
		expectedBody   string
		want           string
	}{
		{
			"returns current database name",
			"test_db",
			"test_db",
			200,
			`{"db_name":"test_db"}`,
			"",
		},
		{
			"returns empty database name",
			"",
			"",
			200,
			`{"db_name":""}`,
			"",
		},
		{
			"returns warning when current db name is different from default",
			"test_db",
			"default_db",
			200,
			`{"db_name":"test_db","warnings":["Current database differs from the default"]}`,
			"Current database differs from the default",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			var output bytes.Buffer

			os.Setenv("DB_NAME", tc.defaultDBName)
			defer os.Unsetenv("DB_NAME")

			dbEnv := &db.Env{Name: tc.dbName}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(&output).Sugar()

			expected := &gabi.Config{DBEnv: dbEnv, Logger: logger}
			GetDBName(expected).ServeHTTP(w, r.WithContext(context.TODO()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			body, err := io.ReadAll(actual.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, actual.StatusCode)
			assert.JSONEq(t, tc.expectedBody, string(body))
			assert.Contains(t, output.String(), tc.want)
		})
	}
}
