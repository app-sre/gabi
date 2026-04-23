package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	gabidb "github.com/app-sre/gabi/pkg/env/db"
)

// discardResponseWriter implements http.ResponseWriter but drops written bytes. Using it in
// benchmarks avoids httptest.ResponseRecorder growing a full in-memory copy of the body, which
// otherwise dominates B/op and obscures the extra [][]string retention in Query for large results.
type discardResponseWriter struct {
	header http.Header
	code   int
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{header: make(http.Header)}
}

func (d *discardResponseWriter) Header() http.Header { return d.header }

func (d *discardResponseWriter) WriteHeader(code int) { d.code = code }

func (d *discardResponseWriter) Write(p []byte) (int, error) { return len(p), nil }

// benchmarkQueryHandlers runs Query or StreamQuery against sqlmock with rowCount result rows.
// Compare BenchmarkQueryLargeResult vs BenchmarkStreamQueryLargeResult with -benchmem: Query retains
// every row in [][]string before encoding; Stream encodes one row at a time.
func benchmarkQueryHandlers(b *testing.B, rowCount int, stream bool) {
	b.Helper()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		db, mock, err := sqlmock.New()
		if err != nil {
			b.Fatal(err)
		}

		rows := sqlmock.NewRows([]string{"c"})
		for range rowCount {
			rows.AddRow("v")
		}
		mock.ExpectBegin()
		mock.ExpectQuery(`select 1;`).WillReturnRows(rows)
		mock.ExpectCommit()

		logger := test.DummyLogger(io.Discard).Sugar()
		cfg := &gabi.Config{
			DB:      db,
			DBEnv:   &gabidb.Env{},
			Logger:  logger,
			Encoder: base64.StdEncoding,
		}

		var h http.HandlerFunc
		if stream {
			h = StreamQuery(cfg)
		} else {
			h = Query(cfg)
		}

		w := newDiscardResponseWriter()
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"query": "select 1;"}`))
		h.ServeHTTP(w, r)

		if err := mock.ExpectationsWereMet(); err != nil {
			b.Fatal(err)
		}
		_ = db.Close()
	}
}

func BenchmarkQueryLargeResult(b *testing.B) {
	for _, n := range []int{100, 2000, 10000} {
		b.Run(fmt.Sprintf("rows_%d", n), func(b *testing.B) {
			benchmarkQueryHandlers(b, n, false)
		})
	}
}

func BenchmarkStreamQueryLargeResult(b *testing.B) {
	for _, n := range []int{100, 2000, 10000} {
		b.Run(fmt.Sprintf("rows_%d", n), func(b *testing.B) {
			benchmarkQueryHandlers(b, n, true)
		})
	}
}
