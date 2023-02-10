package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthcheck(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       func(sqlmock.Sqlmock)
		code        int
		body        string
	}{
		{
			"database is accessible and returns ping reply",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			200,
			`{"status":"OK"}`,
		},
		{
			"database is not accessible",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(errors.New("test"))
			},
			503,
			`{"database":"Unable to connect to the database"}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var body bytes.Buffer

			db, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
			defer func() { _ = db.Close() }()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", &bytes.Buffer{})

			logger := test.DummyLogger(io.Discard).Sugar()

			tc.given(mock)

			expected := &gabi.Env{DB: db, Logger: logger}
			Healthcheck(expected).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			err := mock.ExpectationsWereMet()

			require.NoError(t, err)
			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, body.String(), tc.body)
		})
	}
}
