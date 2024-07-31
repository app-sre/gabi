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

func TestSwitchDBName(t *testing.T) {
	cases := []struct {
		description   string
		initialDBName string
		newDBName     string
		code          int
		body          map[string]string
	}{
		{
			"override database name",
			"initial_db",
			"new_db",
			200,
			map[string]string{"db_name": "new_db"},
		},
		{
			"empty database name",
			"initial_db",
			"",
			200,
			map[string]string{"db_name": ""},
		},
		{
			"invalid request payload",
			"initial_db",
			"",
			400,
			map[string]string{"error": "Invalid request payload"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			var body bytes.Buffer

			dbEnv := &db.Env{Name: tc.initialDBName}

			w := httptest.NewRecorder()
			var requestBody []byte
			if tc.description == "invalid request payload" {
				requestBody = []byte(`invalid payload`)
			} else {
				requestBody, _ = json.Marshal(map[string]string{"db_name": tc.newDBName})
			}
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(requestBody))

			logger := test.DummyLogger(io.Discard).Sugar()

			expected := &gabi.Config{DBEnv: dbEnv, Logger: logger}
			SwitchDBName(expected).ServeHTTP(w, r.WithContext(context.TODO()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			var responseBody map[string]string
			err := json.Unmarshal(body.Bytes(), &responseBody)

			if tc.description == "invalid request payload" {
				require.Error(t, err)
				assert.Equal(t, tc.code, actual.StatusCode)
				assert.Contains(t, body.String(), tc.body["error"])
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.code, actual.StatusCode)
				assert.Equal(t, tc.body, responseBody)
				assert.Equal(t, tc.newDBName, dbEnv.GetCurrentDBName())
			}
		})
	}
}
