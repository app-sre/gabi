package handlers

import (
	"context"
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
					return env.DB.PingContext(ctx)
				},
			),
		),
	)
}
