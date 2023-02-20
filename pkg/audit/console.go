package audit

import (
	"go.uber.org/zap"
)

type ConsoleAudit struct {
	Logger *zap.SugaredLogger
}

var _ Audit = (*ConsoleAudit)(nil)

func NewLoggerAudit(logger *zap.SugaredLogger) *ConsoleAudit {
	return &ConsoleAudit{Logger: logger}
}

func (d *ConsoleAudit) Write(q *QueryData) error {
	d.Logger.Infow("AUDIT",
		"Query", q.Query,
		"User", q.User,
		"Timestamp", q.Timestamp,
	)
	return nil
}
