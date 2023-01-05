package test

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func DummyLogger(w io.Writer) *zap.Logger {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey: "message",
	})

	writer := zap.CombineWriteSyncers(zapcore.AddSync(os.Stderr), zapcore.AddSync(w))

	l := zap.New(zapcore.NewCore(encoder, writer, zapcore.DebugLevel))
	zap.RedirectStdLog(l)

	return l
}
