package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/app-sre/gabi/internal/test"
	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/splunk"
	"github.com/stretchr/testify/assert"
)

func TestAudit(t *testing.T) {
	cases := []struct {
		description string
		given       func(*httptest.Server) *splunk.SplunkEnv
		context     func() context.Context
		headers     func(*bytes.Buffer) func(*http.Request)
		request     func() *bytes.Buffer
		handler     func(*bytes.Buffer) func(http.ResponseWriter, *http.Request)
		code        int
		body        string
		response    string
		output      *regexp.Regexp
	}{
		{
			"valid query",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint:  s.URL,
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			200,
			``,
			`{"query":"select 1;","user":"test","namespace":"test","pod":"test"}`,
			regexp.MustCompile(`AUDIT\s{"Query": "select 1;", "User": "test", "Timestamp": \d{10}}`),
		},
		{
			"valid query with user passed via context",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint:  s.URL,
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func() context.Context {
				ctx := context.TODO()
				return context.WithValue(ctx, contextUserKey, "test2")
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			200,
			``,
			`{"query":"select 1;","user":"test2","namespace":"test","pod":"test"}`,
			regexp.MustCompile(`AUDIT\s{"Query": "select 1;", "User": "test2", "Timestamp": \d{10}}`),
		},
		{
			"valid query with no SQL statements provided",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": ""}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			200,
			``,
			``,
			regexp.MustCompile(`AUDIT\s{"Query": "", "User": "test", "Timestamp": \d{10}}`),
		},
		{
			"valid query with no Splunk endpoint configured",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			500,
			`An internal error has occurred`,
			``,
			regexp.MustCompile(``),
		},
		{
			"valid query with invalid Splunk endpoint configured",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: "http://test",
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			500,
			`An internal error has occurred`,
			``,
			regexp.MustCompile(``),
		},
		{
			"valid query with an error in Splunk response",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code":123,"Text":"test"}`)
				}
			},
			500,
			`An internal error has occurred`,
			``,
			regexp.MustCompile(``),
		},
		{
			"valid query with malformed JSON in Splunk response",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code:0,"Text":""}`)
				}
			},
			500,
			`An internal error has occurred`,
			``,
			regexp.MustCompile(``),
		},
		{
			"invalid query with empty body",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			200,
			``,
			``,
			regexp.MustCompile(`Unable to unmarshal request body`),
		},
		{
			"invalid query with malformed JSON in the body",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
					r.Header.Set("X-Forwarded-User", "test")
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query: "select 1;"}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			200,
			``,
			``,
			regexp.MustCompile(`Unable to unmarshal request body`),
		},
		{
			"invalid query with no required headers set",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					// No-op.
				}
			},
			func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			400,
			`Request without required header: Content-Length`,
			``,
			regexp.MustCompile(``),
		},
		{
			"invalid query with no required user header set",
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func() context.Context {
				return context.TODO()
			},
			func(b *bytes.Buffer) func(r *http.Request) {
				return func(r *http.Request) {
					r.Header.Set("Content-Length", fmt.Sprint(b.Len()))
				}
			},
			func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			400,
			`Request without required header: X-Forwarded-User`,
			``,
			regexp.MustCompile(``),
		},
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var client, server, output bytes.Buffer

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", tc.request())

			s := httptest.NewServer(http.HandlerFunc(tc.handler(&server)))
			defer s.Close()

			logger := test.DummyLogger(&output).Sugar()

			la := &audit.LoggerAudit{Logger: logger}
			sa := &audit.SplunkAudit{SplunkEnv: tc.given(s)}
			sa.SetHTTPClient(http.DefaultClient)

			tc.headers(tc.request())(r)

			aux := &gabi.Env{LoggerAudit: la, SplunkAudit: sa, Logger: logger}
			Audit(aux)(dummyHandler).ServeHTTP(w, r.WithContext(tc.context()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&client, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, client.String(), tc.body)
			assert.Contains(t, server.String(), tc.response)
			assert.Regexp(t, tc.output, output.String())
		})
	}
}
