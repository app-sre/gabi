package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/etherlabsio/healthcheck/v2"

	gabi "github.com/app-sre/gabi/pkg"
)

const healthcheckTimeout = 5 * time.Second

func Healthcheck(cfg *gabi.Config) http.Handler {
	return healthcheck.Handler(
		healthcheck.WithTimeout(healthcheckTimeout),
		healthcheck.WithChecker(
			"database", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					err := cfg.DB.PingContext(ctx)
					if err != nil {
						l := "Unable to connect to the database"
						cfg.Logger.Errorf("%s: %s", l, err)
						return errors.New(l)
					}
					return nil
				},
			),
		),
	)
}
