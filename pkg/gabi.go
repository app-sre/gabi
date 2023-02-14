package gabi

import (
	"database/sql"
	"encoding/base64"
	"os"

	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/env/user"
	"go.uber.org/zap"
)

type Config struct {
	DB          *sql.DB
	DBEnv       *db.Env
	UserEnv     *user.Env
	LoggerAudit audit.Audit
	SplunkAudit audit.Audit
	Logger      *zap.SugaredLogger
	Encoder     *base64.Encoding
}

func Production() bool {
	return os.Getenv("ENVIRONMENT") == "production"
}
