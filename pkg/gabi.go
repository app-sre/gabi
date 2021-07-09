package gabi

import (
	"database/sql"

	"go.uber.org/zap"
)

const Version = "0.0.1"

type Env struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}
