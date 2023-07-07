package audit

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/app-sre/gabi/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerAudit(t *testing.T) {
	t.Parallel()

	logger := test.DummyLogger(io.Discard).Sugar()
	actual := NewLoggerAudit(logger)

	require.NotNil(t, actual)
	assert.IsType(t, &ConsoleAudit{}, actual)
}

func TestLoggingAuditWrite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       QueryData
		want        *regexp.Regexp
	}{
		{
			"query data with all fields set",
			QueryData{Query: "select 1;", User: "test", Timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()},
			regexp.MustCompile(`AUDIT\s{"Query": "select 1;", "User": "test", "Timestamp": 1672531200}`),
		},
		{
			"query data with no SQL statements provided",
			QueryData{Query: "", User: "test", Timestamp: time.Now().Unix()},
			regexp.MustCompile(`AUDIT\s{"Query": "", "User": "test", "Timestamp": \d{10}}`),
		},
		{
			"invalid query data with nothing set",
			QueryData{},
			regexp.MustCompile(`AUDIT\s{"Query": "", "User": "", "Timestamp": 0}`),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer

			logger := test.DummyLogger(&output).Sugar()

			audit := &ConsoleAudit{Logger: logger}
			err := audit.Write(context.TODO(), &tc.given)

			require.NoError(t, err)
			assert.Regexp(t, tc.want, output.String())
		})
	}
}
