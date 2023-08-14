package gabi

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/env/user"
	"go.uber.org/zap"
)

const (
	// The total time it takes to read the request from the client.
	DefaultReadTimeout = 1 * time.Minute

	// The total time it takes to execute the request.
	DefaultRequestTimeout = 2 * time.Minute
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

func RequestTimeout() time.Duration {
	t, err := parseDuration(os.Getenv("REQUEST_TIMEOUT"))
	if err != nil || t == 0 {
		return DefaultRequestTimeout
	}
	return t
}

func parseDuration(duration string) (time.Duration, error) {
	var t time.Duration

	n, err := strconv.ParseInt(duration, 10, 64)
	if err == nil {
		t = time.Duration(n) * time.Second
	} else {
		t, err = time.ParseDuration(duration)
	}
	if err != nil {
		return 0, fmt.Errorf("unable to parse duration: %w", err)
	}

	if t < 0 {
		t = -t
	}

	return t, nil
}
