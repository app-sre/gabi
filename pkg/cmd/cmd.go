package cmd

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"go.uber.org/zap"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/env/splunk"
	"github.com/app-sre/gabi/pkg/env/user"
	"github.com/app-sre/gabi/pkg/handlers"
	"github.com/app-sre/gabi/pkg/middleware"
	"github.com/app-sre/gabi/pkg/version"
)

const (
	readTimeout       = 1 * time.Minute
	readHeaderTimeout = 20 * time.Second
	writeTimeout      = 2 * time.Minute
)

func Run(logger *zap.SugaredLogger) error {
	production := os.Getenv("ENVIRONMENT") == "production"
	logger.Infof("Starting GABI version: %s", version.Version())

	usere := user.NewUserEnv()
	err := usere.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure users: %w", err)
	}

	expiry := usere.IsExpired()
	date := usere.Expiration.Format(user.ExpiryDateLayout)
	if usere.IsDeprecated() {
		expiry = len(usere.Users) == 0
		date = "UNKNOWN"
	}

	logger.Infof("Production: %t, expired: %t (expiration date: %s)", production, expiry, date)
	logger.Debugf("Authorized users: %v", usere.Users)

	dbe := db.NewDBEnv()
	err = dbe.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure database: %w", err)
	}
	logger.Infof("Using database driver: %s (write access: %t)", dbe.Driver, dbe.AllowWrite)

	db, err := sql.Open(dbe.Driver.Name(), dbe.ConnectionDSN())
	if err != nil {
		return fmt.Errorf("unable to open database connection: %w", err)
	}
	defer db.Close()
	logger.Debugf("Connected to database host: %s (port: %d)", dbe.Host, dbe.Port)

	la := audit.NewLoggerAudit(logger)

	se := splunk.NewSplunkEnv()
	err = se.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure Splunk: %w", err)
	}
	logger.Infof("Sending audit to Splunk endpoint: %s", se.Endpoint)

	sa := audit.NewSplunkAudit(se)

	env := &gabi.Env{
		DB:          db,
		DBEnv:       dbe,
		UserEnv:     usere,
		LoggerAudit: la,
		SplunkAudit: sa,
		Logger:      logger,
	}

	// Temp workaround for easy to access io.Writer.
	defaultLogOutput := log.Default().Writer()

	healthLogOutput := io.Discard
	if !production {
		healthLogOutput = defaultLogOutput
	}
	logHandler := gorillaHandlers.LoggingHandler

	queryChain := alice.New(
		alice.Constructor(middleware.Recovery(env)),
		alice.Constructor(middleware.Authorization(env)),
		alice.Constructor(middleware.Expiration(env)),
		alice.Constructor(middleware.Audit(env)),
	).Then(handlers.Query(env))

	r := mux.NewRouter()
	r.Handle("/healthcheck", logHandler(healthLogOutput, handlers.Healthcheck(env))).Methods("GET")
	r.Handle("/query", logHandler(defaultLogOutput, queryChain)).Methods("POST")

	port := 8080
	logger.Infof("HTTP server starting on port: %d", port)

	server := &http.Server{
		Addr:              net.JoinHostPort("", strconv.Itoa(port)),
		Handler:           r,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("unable to start HTTP server: %w", err)
	}

	return nil
}
