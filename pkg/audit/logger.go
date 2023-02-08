package audit

import (
	"go.uber.org/zap"
)

type LoggerAudit struct {
	Logger *zap.SugaredLogger
}

var _ Audit = (*LoggerAudit)(nil)

func NewLoggerAudit(logger *zap.SugaredLogger) *LoggerAudit {
	return &LoggerAudit{Logger: logger}
}

func (d *LoggerAudit) Write(q *QueryData) error {
	d.Logger.Infow("AUDIT",
		"Query", q.Query,
		"User", q.User,
		"Timestamp", q.Timestamp,
	)
	return nil
}
