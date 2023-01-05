package gabi

import (
	"database/sql"

	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/env/user"
	"go.uber.org/zap"
)

type Env struct {
	DB          *sql.DB
	DBEnv       *db.DBEnv
	UserEnv     *user.UserEnv
	LoggerAudit *audit.LoggerAudit
	SplunkAudit *audit.SplunkAudit
	Logger      *zap.SugaredLogger
}
