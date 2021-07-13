package cmd

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
	"go.uber.org/zap"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/handlers"
)

func Run(logger *zap.SugaredLogger) {
	dbe := db.Dbenv{}
	err := dbe.Populate()
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("Database environment variables populated.")

	// Add audit backend selection method via viper / config files
	a := &audit.LoggingAudit{Logger: logger}

	logger.Info("Using default audit backend: stdout logger.")

	logger.Info("Establishing DB connection pool.")
	db, err := sql.Open(dbe.DB_DRIVER, dbe.ConnStr)
	if err != nil {
		logger.Fatal("Fatal error opening database.")
	}
	defer db.Close()
	logger.Info("Database connection handle established.")
	logger.Infof("Using %s database driver.", dbe.DB_DRIVER)

	env := &gabi.Env{DB: db, Logger: logger, Audit: a}

	r := mux.NewRouter()

	r.Handle("/healthcheck", handlers.Healthcheck(env))
	r.HandleFunc("/query", handlers.Query(env))

	logger.Info("Router initialized.")

	servePort := 8080
	logger.Infof("HTTP server starting on port %d.", servePort)

	// Temp workaround for easy to access io.Writer
	httpLogger := log.Default()
	http.ListenAndServe(
		":"+strconv.Itoa(servePort),
		gorillaHandlers.LoggingHandler(httpLogger.Writer(), r),
	)
}
