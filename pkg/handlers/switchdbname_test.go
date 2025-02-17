package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"database/sql"
	"errors"

	"github.com/DATA-DOG/go-sqlmock"
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
		sqlOpener     func(string, string) (*sql.DB, error)
		code          int
		body          map[string]string
	}{
		{
			"override database name",
			"initial_db",
			"new_db",
			func(s string, s2 string) (db *sql.DB, err error) {
				new_db, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
				mock.ExpectPing()
				return new_db, nil
            },
			200,
			map[string]string{"db_name": "new_db"},
		},
		{
			"invalid database name",
			"initial_db",
			"invalid_db",
			func(s string, s2 string) (db *sql.DB, err error) {
				new_db, _, _ := sqlmock.New()				
				return new_db, errors.New("connection refused")
            },
			400,
			map[string]string{"error": "Unable to open database connection"},
		},
		{
			"invalid request payload",
			"initial_db",
			"",
			func(s string, s2 string) (db *sql.DB, err error) {
				return nil, nil
            },
			400,
			map[string]string{"error": "Invalid request payload"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			var body bytes.Buffer

			dbEnv := &db.Env{Name: tc.initialDBName}
			db, _, _ := sqlmock.New()

			w := httptest.NewRecorder()
			var requestBody []byte
			if tc.description == "invalid request payload" {
				requestBody = []byte(`invalid payload`)
			} else {
				requestBody, _ = json.Marshal(map[string]string{"db_name": tc.newDBName})
			}
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(requestBody))

			logger := test.DummyLogger(io.Discard).Sugar()

			expected := &gabi.Config{DB: db, DBEnv: dbEnv, Logger: logger}
			defer func() { _ = expected.DB.Close() }()

			gabi.SQLOpen = tc.sqlOpener
			SwitchDBName(expected).ServeHTTP(w, r.WithContext(context.TODO()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			var responseBody map[string]string
			err := json.Unmarshal(body.Bytes(), &responseBody)

			if tc.description == "override database name" {
				require.NoError(t, err)
				assert.Equal(t, tc.code, actual.StatusCode)
				assert.Equal(t, tc.body, responseBody)
				assert.Equal(t, tc.newDBName, expected.GetCurrentDBName())
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.code, actual.StatusCode)
				assert.Contains(t, body.String(), tc.body["error"])
			}
		})
	}
}
