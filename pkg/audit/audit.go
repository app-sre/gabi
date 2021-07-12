package audit

import (
	"time"

	gabi "github.com/app-sre/gabi/pkg"
)

// stdout logging (json format, with "query", "user", "ts" values).
type Audit interface {
	write(QueryMetadata) error
}

type QueryMetadata struct {
	timestamp time.Time
	query     string
	user      string
}

func LogAudit(env *gabi.Env, q *QueryMetadata) error {

}

func CloudwatchAudit() error
