package gabi

import (
	"database/sql"
	"os"

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

func Production() bool {
	return os.Getenv("ENVIRONMENT") == "production"
}
