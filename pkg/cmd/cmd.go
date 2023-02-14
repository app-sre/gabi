package cmd

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

	logger.Infof("Production: %t, expired: %t (expiration date: %s)", gabi.Production(), expiry, date)
	logger.Debugf("Authorized users: %v", usere.Users)

	dbe := db.NewDBEnv()
	err = dbe.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure database: %w", err)
	}
	logger.Infof("Using database driver: %s (write access: %t)", dbe.Driver, dbe.AllowWrite)

	db, err := sql.Open(dbe.Driver.String(), dbe.ConnectionDSN())
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

	cfg := &gabi.Config{
		DB:          db,
		DBEnv:       dbe,
		UserEnv:     usere,
		LoggerAudit: la,
		SplunkAudit: sa,
		Logger:      logger,
		Encoder:     base64.StdEncoding,
	}

	// Temp workaround for easy to access io.Writer.
	defaultLogOutput := log.Default().Writer()

	healthLogOutput := io.Discard
	if !gabi.Production() {
		healthLogOutput = defaultLogOutput
	}
	logHandler := gorillaHandlers.LoggingHandler

	queryChain := alice.New(
		alice.Constructor(middleware.Recovery(cfg)),
		alice.Constructor(middleware.Authorization(cfg)),
		alice.Constructor(middleware.Expiration(cfg)),
		alice.Constructor(middleware.Audit(cfg)),
	)
	queryHandler := queryChain.Then(handlers.Query(cfg))

	r := mux.NewRouter()
	r.Handle("/healthcheck", logHandler(healthLogOutput, handlers.Healthcheck(cfg))).Methods("GET")
	r.Handle("/query", logHandler(defaultLogOutput, queryHandler)).Methods("POST")

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
