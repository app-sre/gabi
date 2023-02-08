package audit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/app-sre/gabi/pkg/env/splunk"
	"github.com/app-sre/gabi/pkg/version"
	"github.com/stretchr/testify/assert"
)

func TestNewSplunkAudit(t *testing.T) {
	cases := []struct {
		description string
		expected    *splunk.SplunkEnv
		given       Option
		option      bool
	}{
		{
			"using option that updates internal state",
			&splunk.SplunkEnv{Index: "test"},
			func(s *SplunkAudit) {
				s.SplunkEnv.Index = "test"
			},
			true,
		},
		{
			"using option that does nothing",
			&splunk.SplunkEnv{},
			func(s *SplunkAudit) {
				// No-op.
			},
			true,
		},
		{
			"without using any options",
			&splunk.SplunkEnv{},
			func(s *SplunkAudit) {
				// No-op.
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var actual *SplunkAudit

			if tc.option {
				actual = NewSplunkAudit(&splunk.SplunkEnv{}, tc.given)
			} else {
				actual = NewSplunkAudit(&splunk.SplunkEnv{})
			}

			assert.NotNil(t, actual)
			assert.IsType(t, &SplunkAudit{}, actual)
			assert.Equal(t, tc.expected, actual.SplunkEnv)
		})
	}

}

func TestWithHTTPClient(t *testing.T) {
	cases := []struct {
		description string
		given       []Option
		defaults    bool
	}{
		{
			"using default HTTP client set internally",
			[]Option{},
			true,
		},
		{
			"using custom HTTP client",
			[]Option{WithHTTPClient(http.DefaultClient)},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			aux := NewSplunkAudit(&splunk.SplunkEnv{}, tc.given...)

			actual := aux.client

			if tc.defaults {
				assert.NotNil(t, actual.Transport)
			} else {
				assert.Nil(t, actual.Transport)
			}

			assert.NotNil(t, aux)
			assert.IsType(t, &SplunkAudit{}, aux)
		})
	}

}

func TestSetHTTPClient(t *testing.T) {
	cases := []struct {
		description string
		given       func(*SplunkAudit)
		defaults    bool
	}{
		{
			"using default HTTP client set internally",
			func(s *SplunkAudit) {
				// No-op.
			},
			true,
		},
		{
			"using custom HTTP client",
			func(s *SplunkAudit) {
				s.SetHTTPClient(http.DefaultClient)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			aux := NewSplunkAudit(&splunk.SplunkEnv{})
			tc.given(aux)

			actual := aux.client

			if tc.defaults {
				assert.NotNil(t, actual.Transport)
			} else {
				assert.Nil(t, actual.Transport)
			}

			assert.NotNil(t, aux)
			assert.NotNil(t, actual)
			assert.IsType(t, &SplunkAudit{}, aux)
		})
	}
}

func TestSplunkAduitWrite(t *testing.T) {
	cases := []struct {
		description string
		given       QueryData
		headers     func() *http.Header
		server      func(*httptest.Server) *splunk.SplunkEnv
		handler     func(*bytes.Buffer, *http.Header) func(w http.ResponseWriter, r *http.Request)
		error       bool
		message     string
		output      *regexp.Regexp
	}{
		{
			"valid query",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk test123"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint:  s.URL,
					Token:     "test123",
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			false,
			``,
			regexp.MustCompile(`{"query":"select 1;","user":"test","namespace":"test","pod":"test"},(.*),"time":1672531200`),
		},
		{
			"valid query with no SQL statements provided",
			QueryData{Query: "", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk test123"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint:  s.URL,
					Token:     "test123",
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			false,
			``,
			regexp.MustCompile(`{"query":"","user":"test","namespace":"test","pod":"test"},(.*),"time":\d{10}`),
		},
		{
			"valid query with invalid Splunk environment set",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			false,
			``,
			regexp.MustCompile(`{"query":"select 1;","user":"test","namespace":"","pod":""},(.*),"time":\d{10}`),
		},
		{
			"valid query with no Splunk endpoint configured",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			true,
			`unable to send request to Splunk`,
			regexp.MustCompile(``),
		},
		{
			"valid query with invalid Splunk endpoint configured",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: "http://test",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			true,
			`unable to send request to Splunk`,
			regexp.MustCompile(``),
		},
		{
			"valid query with an error in Splunk response",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk test123"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
					Token:    "test123",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code":123,"Text":"test"}`)
				}
			},
			true,
			`unable to write to Splunk`,
			regexp.MustCompile(``),
		},
		{
			"valid query with malformed JSON in Splunk response",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Now().Unix()},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk test123"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint: s.URL,
					Token:    "test123",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code:0,"Text":""}`)
				}
			},
			true,
			`unable to unmarshal Splunk response`,
			regexp.MustCompile(``),
		},
		{
			"invalid query and invalid Splunk environment set",
			QueryData{},
			func() *http.Header {
				return &http.Header{}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					// No-op.
				}
			},
			true,
			`unable to send request to Splunk`,
			regexp.MustCompile(``),
		},
		{
			"invalid query data with nothing set",
			QueryData{},
			func() *http.Header {
				return &http.Header{
					"Accept":          []string{"application/json"},
					"Accept-Encoding": []string{"gzip"},
					"Authorization":   []string{"Splunk test123"},
					"Content-Type":    []string{"application/json; charset=utf-8"},
					"User-Agent":      []string{fmt.Sprintf("GABI/%s", version.Version())},
				}
			},
			func(s *httptest.Server) *splunk.SplunkEnv {
				return &splunk.SplunkEnv{
					Endpoint:  s.URL,
					Token:     "test123",
					Host:      "test",
					Namespace: "test",
					Pod:       "test",
				}
			},
			func(b *bytes.Buffer, h *http.Header) func(w http.ResponseWriter, r *http.Request) {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(b, r.Body)
					*h = r.Header
					h.Del("Content-Length")
					fmt.Fprintln(w, `{"Code":0,"Text":""}`)
				}
			},
			false,
			``,
			regexp.MustCompile(`{"query":"","user":"","namespace":"test","pod":"test"},(.*),"time":0`),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var server bytes.Buffer

			headers := make(http.Header)

			s := httptest.NewServer(http.HandlerFunc(tc.handler(&server, &headers)))
			defer s.Close()

			actual := &SplunkAudit{SplunkEnv: tc.server(s)}
			actual.SetHTTPClient(http.DefaultClient)
			err := actual.Write(&tc.given)

			if tc.error {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.message)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.headers(), &headers)
			assert.Regexp(t, tc.output, server.String())
		})
	}
}
