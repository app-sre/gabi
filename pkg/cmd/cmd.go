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

	gorillahandlers "github.com/gorilla/handlers"
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

func Run(logger *zap.SugaredLogger) error {
	logger.Infof("Starting GABI version: %s", version.Version())

	usere := user.NewUserEnv()
	err := usere.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure users: %w", err)
	}

	expiry := usere.IsExpired()
	date := usere.Expiration.Format(user.ExpiryDateLayout)
	logger.Infof("Production: %t, expired: %t (expiration date: %s)", gabi.Production(), expiry, date)
	logger.Debugf("Authorized users: %v", usere.Users)

	dbe := db.NewDBEnv()
	err = dbe.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure database: %w", err)
	}
	logger.Infof("Using database driver: %s (write access: %t)", dbe.Driver, dbe.AllowWrite)

	db, err := sql.Open(dbe.Driver.String(), dbe.ConnectionDSN(""))
	if err != nil {
		return fmt.Errorf("unable to open database connection: %w", err)
	}
	logger.Debugf("Connected to database host: %s (port: %d)", dbe.Host, dbe.Port)

	se := splunk.NewSplunkEnv()
	err = se.Populate()
	if err != nil {
		return fmt.Errorf("unable to configure Splunk: %w", err)
	}
	logger.Infof("Sending audit to Splunk endpoint: %s", se.Endpoint)

	cfg := &gabi.Config{
		DB:          db,
		DBEnv:       dbe,
		UserEnv:     usere,
		LoggerAudit: audit.NewLoggerAudit(logger),
		SplunkAudit: audit.NewSplunkAudit(se),
		Logger:      logger,
		Encoder:     base64.StdEncoding,
	}
	defer cfg.DB.Close()
	timeout := gabi.RequestTimeout()

	// Temporary workaround for easy to access io.Writer.
	defaultLogOutput := log.Default().Writer()

	healthLogOutput := io.Discard
	if !gabi.Production() {
		healthLogOutput = defaultLogOutput
	}
	logHandler := gorillahandlers.LoggingHandler

	queryChain := alice.New(
		alice.Constructor(middleware.Recovery(cfg)),
		alice.Constructor(middleware.Authorization(cfg)),
		alice.Constructor(middleware.Expiration(cfg)),
		alice.Constructor(middleware.Audit(cfg)),
		alice.Constructor(middleware.Timeout(timeout)),
	)
	queryHandler := queryChain.Then(handlers.Query(cfg))

	r := mux.NewRouter()
	r.Handle("/healthcheck", logHandler(healthLogOutput, handlers.Healthcheck(cfg))).Methods("GET")
	r.Handle("/query", logHandler(defaultLogOutput, queryHandler)).Methods("POST")
	r.Handle("/dbname", logHandler(defaultLogOutput, handlers.GetCurrentDBName(cfg))).Methods("GET")
	r.Handle("/dbname/switch", logHandler(defaultLogOutput, handlers.SwitchDBName(cfg))).Methods("POST")

	port := 8080
	logger.Infof("HTTP server starting on port: %d", port)

	server := &http.Server{
		Addr:        net.JoinHostPort("", strconv.Itoa(port)),
		Handler:     r,
		ReadTimeout: gabi.DefaultReadTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("unable to start HTTP server: %w", err)
	}

	return nil
}
