package audit

import (
	"go.uber.org/zap"
)

type QueryData struct {
	Query     string
	User      string
	Timestamp int64
}

type Audit interface {
	Write(*QueryData) error
}

type LoggingAudit struct {
	Logger *zap.SugaredLogger
}

func (d *LoggingAudit) Write(q *QueryData) error {
	d.Logger.Infow("gabi API audit record",
		"Query", q.Query,
		"User", q.User,
		"Timestamp", q.Timestamp,
	)
	return nil
}

type CloudwatchAudit struct{}

func (c *CloudwatchAudit) Write(q *QueryData) error {
	return nil
}
