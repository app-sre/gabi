package cmd

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/etherlabsio/healthcheck/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4"
	"go.uber.org/zap"

	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/query"
)

func Run() {
	dbe := db.Dbenv{}
	err := dbe.Populate()
	if err != nil {
		log.Fatal(err)
	}
	zap.S().Info("Database environment variables populated.")

	DB, err := sql.Open(dbe.DB_DRIVER, dbe.ConnStr)
	if err != nil {
		log.Fatal("Fatal error opening database.")
	}
	defer DB.Close()
	zap.S().Info("Database connection handle established.")
	zap.S().Infof("Using %s database driver.", dbe.DB_DRIVER)

	r := mux.NewRouter()

	r.Handle("/healthcheck", healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker(
			"database", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					return DB.PingContext(ctx)
				},
			),
		),
	))

	r.HandleFunc("/query", query.Handler)
	zap.S().Info("Router initialized.")

	servePort := 8080
	zap.S().Infof("HTTP server starting on port %d.", servePort)

	// Temp workaround for easy to access io.Writer
	httpLogger := log.Default()
	http.ListenAndServe(
		":"+strconv.Itoa(servePort),
		handlers.LoggingHandler(httpLogger.Writer(), r),
	)
}
