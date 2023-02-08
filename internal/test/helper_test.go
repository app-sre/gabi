package test

import (
	"bytes"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDummyLogger(t *testing.T) {
	cases := []struct {
		description string
		given       func(*zap.Logger)
		writer      func() io.Writer
		reader      func(io.Writer) string
		output      string
	}{
		{
			"capture logs into a buffer from Zap",
			func(l *zap.Logger) {
				l.Info("test")
			},
			func() io.Writer {
				return &bytes.Buffer{}
			},
			func(w io.Writer) string {
				return w.(*bytes.Buffer).String()
			},
			`test`,
		},
		{
			"capture logs into a buffer redirected from default Go log package",
			func(l *zap.Logger) {
				log.Println("test")
			},
			func() io.Writer {
				return &bytes.Buffer{}
			},
			func(w io.Writer) string {
				return w.(*bytes.Buffer).String()
			},
			`test`,
		},
		{
			"capture logs info a buffer from Zap and default Go log package",
			func(l *zap.Logger) {
				l.Info("test")
				log.Println("test2")
			},
			func() io.Writer {
				return &bytes.Buffer{}
			},
			func(w io.Writer) string {
				return w.(*bytes.Buffer).String()
			},
			"test\ntest2",
		},
		{
			"capture logs from Zap and discard the content",
			func(l *zap.Logger) {
				l.Info("test")
			},
			func() io.Writer {
				return io.Discard
			},
			func(w io.Writer) string {
				return ""
			},
			``,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			w := tc.writer()

			actual := DummyLogger(w)

			tc.given(actual)
			s := tc.reader(w)

			assert.NotNil(t, actual)
			assert.IsType(t, &zap.Logger{}, actual)
			assert.Contains(t, s, tc.output)
		})
	}

}
