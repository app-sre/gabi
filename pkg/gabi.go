package gabi

import (
	"database/sql"

	"github.com/app-sre/gabi/pkg/audit"
	"go.uber.org/zap"
)

const Version = "0.0.1"

type Env struct {
	DB          *sql.DB
	DB_WRITE    bool
	Logger      *zap.SugaredLogger
	Audit       audit.Audit
	SplunkAudit audit.SplunkAudit
	Users       []string
}
