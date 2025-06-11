package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	gabidb "github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/middleware"
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
		context     func() context.Context
		parameters  func(*http.Request)
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"1\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid query with SQL statements passed via context",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"?column?"}).AddRow("2")
				mock.ExpectBegin()
				mock.ExpectQuery(`select 2;`).WillReturnRows(rows)
				mock.ExpectCommit()
			},
			func() context.Context {
				ctx := context.TODO()
				return context.WithValue(ctx, middleware.ContextKeyQuery, "select 2;")
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"2\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid query with empty context value provided",
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
			func() context.Context {
				ctx := context.TODO()
				return context.WithValue(ctx, middleware.ContextKeyQuery, "")
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"1\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid Base64-encoded query",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				q := r.URL.Query()
				q.Add("base64_query", "true")
				r.URL.RawQuery = q.Encode()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "c2VsZWN0IDE7"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"1\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid query with empty HTTP query parameters provided",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				q := r.URL.Query()
				q.Add("base64_query", "")
				r.URL.RawQuery = q.Encode()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"1\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid query with Base64-encoded results",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				q := r.URL.Query()
				q.Add("base64_results", "true")
				r.URL.RawQuery = q.Encode()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"MQ==\"]\n],\"error\":\"\"}",
			``,
		},
		{
			"valid query without Base64-encoded results with empty HTTP query parameters provided",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				q := r.URL.Query()
				q.Add("base64_results", "")
				r.URL.RawQuery = q.Encode()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			200,
			"{\"result\":[[\"?column?\"]\n,[\"1\"]\n],\"error\":\"\"}",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": ""}`)
			},
			200,
			"{\"result\":[null\n],\"error\":\"\"}",
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select * from test;"}`)
			},
			400,
			``,
			`Unable to process database rows: test`,
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
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
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				// No-op.
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": -1}`)
			},
			400,
			``,
			`Unable to decode request body`,
		},
		{
			"invalid query with malformed Base64-encoded value in the body",
			func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				return db, mock
			},
			func(mock sqlmock.Sqlmock) {
				// No-op.
			},
			func() context.Context {
				return context.TODO()
			},
			func(r *http.Request) {
				q := r.URL.Query()
				q.Add("base64_query", "true")
				r.URL.RawQuery = q.Encode()
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "dGhpcyBpcyBhIHRlc3Q=="}`)
			},
			400,
			`Unable to decode Base64-encoded query`,
			``,
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
			encoder := base64.StdEncoding

			db, mock := tc.database()
			defer func() { _ = db.Close() }()

			tc.mock(mock)
			tc.parameters(r)

			expected := &gabi.Config{DB: db, DBEnv: &gabidb.Env{}, Logger: logger, Encoder: encoder}
			Query(expected).ServeHTTP(w, r.WithContext(tc.context()))

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
