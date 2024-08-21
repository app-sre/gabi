package handlers

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/etherlabsio/healthcheck/v2"

	gabi "github.com/app-sre/gabi/pkg"
)

const healthcheckTimeout = 5 * time.Second

func Healthcheck(cfg *gabi.Config) http.Handler {
	defaultDBName := os.Getenv("DB_NAME")
	return healthcheck.Handler(
		healthcheck.WithTimeout(healthcheckTimeout),
		healthcheck.WithChecker(
			"database", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					dbName := cfg.DBEnv.GetCurrentDBName()
					err := cfg.DB.PingContext(ctx)
					if err != nil {
						l := "Unable to connect to the database"
						cfg.Logger.Errorf("%s: %s", l, err)
						return errors.New(l)
					}

					if dbName != defaultDBName {
						l := "Current database differs from the default"
						cfg.Logger.Warnf(l)
					}

					return nil
				},
			),
		),
	)
}
