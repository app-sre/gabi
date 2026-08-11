package gabi

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"sync"
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

	// MaxRequestBodyBytes caps JSON request bodies buffered by middleware and
	// handlers (SQL query payloads, db-switch requests). Sized for legitimate
	// break-glass SQL while preventing memory-exhaustion DoS against the
	// typically single-replica pod (FIND-005).
	MaxRequestBodyBytes = 1 << 20 // 1 MiB

	// MaxHeaderBytes limits request header size on the HTTP server.
	MaxHeaderBytes = 1 << 20 // 1 MiB
)

type Config struct {
	DB          *sql.DB
	DBEnv       *db.Env
	UserEnv     *user.Env
	LoggerAudit audit.Audit
	SplunkAudit audit.Audit
	Logger      *zap.SugaredLogger
	Encoder     *base64.Encoding
	sync.Mutex
}

var (
	SQLOpen = sql.Open
)

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

func (c *Config) OverrideDBName(dbName string) error {
	if err := db.ValidateDBName(dbName); err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()
	dbConn, err := SQLOpen(c.DBEnv.Driver.String(), c.DBEnv.ConnectionDSN(dbName))
	if err != nil {
		return err
	}
	err = dbConn.Ping()
	if err != nil {
		_ = dbConn.Close()
		return err
	}
	c.Logger.Debugf("Connected to database host: %s (dbname: %s)", c.DBEnv.Host, dbName)
	_ = c.DB.Close()
	c.DB = dbConn
	c.DBEnv.Name = dbName
	return nil
}

func (c *Config) GetCurrentDBName() string {
	c.Lock()
	defer c.Unlock()
	return c.DBEnv.Name
}
