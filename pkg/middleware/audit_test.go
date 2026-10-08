package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
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
	t.Parallel()

	cases := []struct {
		description string
		given       func(*httptest.Server) *splunk.Env
		context     func() context.Context
		headers     func(*bytes.Buffer) func(*http.Request)
		request     func() *bytes.Buffer
		handler     func(*bytes.Buffer) func(http.ResponseWriter, *http.Request)
		code        int
		body        string
		response    string
		want        *regexp.Regexp
		query       string
	}{
		{
			"valid query",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			`select 1;`,
		},
		{
			"valid Base64-encoded query",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
					q := r.URL.Query()
					q.Add("base64_query", "true")
					r.URL.RawQuery = q.Encode()
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "c2VsZWN0IDE7"}`)
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
			`select 1;`,
		},
		{
			"valid query with user passed via context",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
					Endpoint:  s.URL,
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func() context.Context {
				ctx := context.TODO()
				return context.WithValue(ctx, ContextKeyUser, "test2")
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
			`select 1;`,
		},
		{
			"valid query with empty HTTP query parameters provided",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
					q := r.URL.Query()
					q.Add("base64_query", "")
					r.URL.RawQuery = q.Encode()
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
			`select 1;`,
		},
		{
			"valid query with no SQL statements provided",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"valid query with no Splunk endpoint configured",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{}
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
			``,
		},
		{
			"valid query with invalid Splunk endpoint configured",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"valid query with an error in Splunk response",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"valid query with malformed JSON in Splunk response",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"invalid query with empty body",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"invalid query with malformed JSON in the body",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"invalid query with no required headers set",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"invalid query with no required user header set",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
			``,
		},
		{
			"reject oversized request body",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
				// Just over MaxRequestBodyBytes (1 MiB).
				return bytes.NewBuffer(bytes.Repeat([]byte("a"), int(gabi.MaxRequestBodyBytes)+1))
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// Must not run.
				}
			},
			http.StatusRequestEntityTooLarge,
			`Request body too large`,
			``,
			regexp.MustCompile(``),
			``,
		},
		{
			"invalid query with malformed Base64-encoded value in the body",
			func(s *httptest.Server) *splunk.Env {
				return &splunk.Env{
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
					q := r.URL.Query()
					q.Add("base64_query", "true")
					r.URL.RawQuery = q.Encode()
				}
			},
			func() *bytes.Buffer {
				return bytes.NewBufferString(`{"query": "dGhpcyBpcyBhIHRlc3Q=="}`)
			},
			func(b *bytes.Buffer) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			400,
			`Unable to decode Base64-encoded query`,
			``,
			regexp.MustCompile(``),
			``,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var (
				client, server bytes.Buffer
				output         bytes.Buffer
				query          string
			)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", tc.request())

			s := httptest.NewServer(http.HandlerFunc(tc.handler(&server)))
			defer s.Close()

			logger := test.DummyLogger(&output).Sugar()
			encoder := base64.StdEncoding

			la := &audit.ConsoleAudit{Logger: logger}
			sa := &audit.SplunkAudit{SplunkEnv: tc.given(s)}
			sa.SetHTTPClient(http.DefaultClient)

			tc.headers(tc.request())(r)

			expected := &gabi.Config{LoggerAudit: la, SplunkAudit: sa, Logger: logger, Encoder: encoder}
			Audit(expected)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query, _ = r.Context().Value(ContextKeyQuery).(string)
			})).ServeHTTP(w, r.WithContext(tc.context()))

			actual := w.Result()
			defer func() { _ = actual.Body.Close() }()

			_, _ = io.Copy(&client, actual.Body)

			assert.Equal(t, tc.code, actual.StatusCode)
			assert.Contains(t, client.String(), tc.body)
			assert.Contains(t, server.String(), tc.response)
			assert.Regexp(t, tc.want, output.String())
			assert.Regexp(t, tc.query, query)
		})
	}
}

func TestAuditLogsRedactedSplunkFailure(t *testing.T) {
	var output bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "<html>WAF private response</html>")
	}))
	defer server.Close()

	logger := test.DummyLogger(&output).Sugar()
	cfg := &gabi.Config{
		Logger:      logger,
		LoggerAudit: &audit.ConsoleAudit{Logger: logger},
		SplunkAudit: audit.NewSplunkAudit(&splunk.Env{Endpoint: server.URL + "/private?token=secret"}),
		Encoder:     base64.StdEncoding,
	}

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"select 1"}`))
	req.Header.Set("Content-Length", fmt.Sprint(req.ContentLength))
	req.Header.Set("X-Forwarded-User", "test")
	response := httptest.NewRecorder()
	handlerCalled := false

	Audit(cfg)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalled = true
	})).ServeHTTP(response, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Contains(t, response.Body.String(), "An internal error has occurred")
	assert.Contains(t, output.String(), "Splunk audit failure; contact help-itde-monitoring-logging")
	assert.Contains(t, output.String(), "received HTTP 403")
	assert.Contains(t, output.String(), "text/html")
	assert.NotContains(t, output.String(), "WAF private response")
	assert.NotContains(t, output.String(), "token=secret")
}
