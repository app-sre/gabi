package handlers

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	gabidb "github.com/app-sre/gabi/pkg/env/db"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		database    func() (*sql.DB, sqlmock.Sqlmock)
		mock        func(sqlmock.Sqlmock)
		request     func() *bytes.Buffer
		code        int
		body        string
		want        string
	}{
		{
			"valid query",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"?column?"}).AddRow("1")
				mock.ExpectBegin()
				mock.ExpectQuery(`select 1;`).WillReturnRows(rows)
				mock.ExpectCommit()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			`{"result":[["?column?"],["1"]],"error":""}`,
			``,
		},
		{
			"valid query with no SQL statements provided",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{})
				mock.ExpectBegin()
				mock.ExpectQuery(``).WillReturnRows(rows)
				mock.ExpectCommit()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": ""}`)
			},
			200,
			`{"result":[null],"error":""}`,
			``,
		},
		{
			"valid query for which database returned query error",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`select 1;`).WillReturnError(errors.New("test"))
				mock.ExpectRollback()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query":"select 1;"}`)
			},
			400,
			`{"result":null,"error":"test"}`,
			``,
		},
		{
			"valid query for which database added a log entry",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`select 1;`).WillReturnError(errors.New("test"))
				mock.ExpectRollback()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query":"select 1;"}`)
			},
			400,
			``,
			`Unable to query database: test`,
		},
		{
			"valid query for which database returned transaction error",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("test"))
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query":"select 1;"}`)
			},
			400,
			``,
			`Unable to start database transaction: test`,
		},
		{
			"valid query for which database returned commit error",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"?column?"}).AddRow("1")
				mock.ExpectBegin()
				mock.ExpectQuery(`select 1;`).WillReturnRows(rows)
				mock.ExpectCommit().WillReturnError(errors.New("test"))
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			400,
			``,
			`Unable to commit database changes: test`,
		},
		{
			"valid query for which database returned query row error",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"?column?"})
				rows.AddRow("1")
				rows.AddRow("2").RowError(1, errors.New("test"))
				mock.ExpectBegin()
				mock.ExpectQuery(`select \* from test;`).WillReturnRows(rows)
				mock.ExpectRollback()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select * from test;"}`)
			},
			400,
			``,
			`Unable to process database query: test`,
		},
		{
			"valid query with database connection error",
			func() (*sql.DB, sqlmock.Sqlmock) {
				_, mock, _ := sqlmock.New()
				db, _ := sql.Open("pgx", "postgres://")
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			503,
			`Unable to connect to the database`,
			``,
		},
		{
			"invalid query with empty body",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				// No-op.
			},
			func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
			400,
			`Request body cannot be empty`,
			``,
		},
		{
			"invalid query with malformed JSON in the body",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query: "select 1;"}`)
			},
			400,
			``,
			`Unable to decode request body`,
		},
		{
			"invalid query with incorrect type provided in the body",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": -1}`)
			},
			400,
			``,
			`Unable to decode request body`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var body, output bytes.Buffer

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", tc.request())

			logger := test.DummyLogger(&output).Sugar()

			db, mock := tc.database()
			defer func() { _ = db.Close() }()

			tc.mock(mock)

			expected := &gabi.Env{DB: db, Logger: logger, DBEnv: &gabidb.DBEnv{}}
			Query(expected).ServeHTTP(w, r)

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&body, actual.Body)

			err := mock.ExpectationsWereMet()

			require.NoError(t, err)
			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, body.String(), tc.body)
			assert.Contains(t, output.String(), tc.want)
		})
	}
}
