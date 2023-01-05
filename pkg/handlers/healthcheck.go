package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/etherlabsio/healthcheck/v2"

	gabi "github.com/app-sre/gabi/pkg"
)

func Healthcheck(env *gabi.Env) http.Handler {
	return healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
		healthcheck.WithChecker(
			"database", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					err := env.DB.PingContext(ctx)
					if err != nil {
						l := "Unable to connect to the database"
						env.Logger.Errorf("%s: %s", l, err)
						return errors.New(l)
					}
					return nil
				},
			),
		),
	)
}
