package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopAudit struct{}

func (nopAudit) Write(context.Context, *audit.QueryData) error { return nil }

type errAudit struct{ err error }

func (a errAudit) Write(context.Context, *audit.QueryData) error { return a.err }

func TestSwitchDBName(t *testing.T) {
	cases := []struct {
		description string
		initialDBName string
		newDBName     string
		rawBody       []byte
		sqlOpener     func(string, string) (*sql.DB, error)
		splunkAudit   audit.Audit
		withUser      bool
		code          int
		body          map[string]string
	}{
		{
			description:   "override database name",
			initialDBName: "initial_db",
			newDBName:     "new_db",
			sqlOpener: func(string, string) (*sql.DB, error) {
				newDB, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
				mock.ExpectPing()
				return newDB, nil
			},
			splunkAudit: nopAudit{},
			withUser:    true,
			code:        200,
			body:        map[string]string{"db_name": "new_db"},
		},
		{
			description:   "connection failure",
			initialDBName: "initial_db",
			newDBName:     "other_db",
			sqlOpener: func(string, string) (*sql.DB, error) {
				return nil, errors.New("connection refused")
			},
			splunkAudit: nopAudit{},
			withUser:    true,
			code:        400,
			body:        map[string]string{"error": "Unable to open database connection"},
		},
		{
			description:   "invalid request payload",
			initialDBName: "initial_db",
			rawBody:       []byte(`invalid payload`),
			sqlOpener:     func(string, string) (*sql.DB, error) { return nil, nil },
			splunkAudit:   nopAudit{},
			withUser:      true,
			code:          400,
			body:          map[string]string{"error": "Invalid request payload"},
		},
		{
			description:   "reject DSN injection in db_name",
			initialDBName: "initial_db",
			newDBName:     "evil?host=attacker.example",
			sqlOpener: func(string, string) (*sql.DB, error) {
				t.Fatal("SQLOpen must not be called for invalid db_name")
				return nil, nil
			},
			splunkAudit: nopAudit{},
			withUser:    true,
			code:        400,
			body:        map[string]string{"error": "Invalid database name"},
		},
		{
			description:   "reject missing authorized user",
			initialDBName: "initial_db",
			newDBName:     "new_db",
			sqlOpener:     func(string, string) (*sql.DB, error) { return nil, nil },
			splunkAudit:   nopAudit{},
			withUser:      false,
			code:          401,
			body:          map[string]string{"error": "Request cannot be authorized"},
		},
		{
			description:   "fail closed when Splunk audit fails",
			initialDBName: "initial_db",
			newDBName:     "new_db",
			sqlOpener: func(string, string) (*sql.DB, error) {
				t.Fatal("SQLOpen must not be called when audit fails")
				return nil, nil
			},
			splunkAudit: errAudit{err: errors.New("splunk down")},
			withUser:    true,
			code:        500,
			body:        map[string]string{"error": "An internal error has occurred"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			var body bytes.Buffer

			dbEnv := &db.Env{Name: tc.initialDBName, Driver: "pgx", Host: "localhost", Port: 5432}
			sqlDB, _, _ := sqlmock.New()

			w := httptest.NewRecorder()
			requestBody := tc.rawBody
			if requestBody == nil {
				requestBody, _ = json.Marshal(map[string]string{"db_name": tc.newDBName})
			}
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(requestBody))
			ctx := context.TODO()
			if tc.withUser {
				ctx = context.WithValue(ctx, middleware.ContextKeyUser, "test-user")
			}
			r = r.WithContext(ctx)

			logger := test.DummyLogger(io.Discard).Sugar()
			expected := &gabi.Config{
				DB:          sqlDB,
				DBEnv:       dbEnv,
				Logger:      logger,
				LoggerAudit: nopAudit{},
				SplunkAudit: tc.splunkAudit,
			}
			defer func() { _ = expected.DB.Close() }()

			gabi.SQLOpen = tc.sqlOpener
			SwitchDBName(expected).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)
			assert.Equal(t, tc.code, actual.StatusCode)

			if tc.code == 200 {
				var responseBody map[string]string
				err := json.Unmarshal(body.Bytes(), &responseBody)
				require.NoError(t, err)
				assert.Equal(t, tc.body, responseBody)
				assert.Equal(t, tc.newDBName, expected.GetCurrentDBName())
				return
			}

			assert.Contains(t, body.String(), tc.body["error"])
			assert.Equal(t, tc.initialDBName, expected.GetCurrentDBName())
		})
	}
}
