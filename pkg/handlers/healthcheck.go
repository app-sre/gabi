package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
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
						errStr := "failed to connect to database as part of healthcheck ping"
						logErr := fmt.Errorf(errStr+": %v", err)
						log.Println(logErr)
						return errors.New("failed to connect database... see gabi logs for further details")
					}
					return nil // healthcheck passed successfully
				},
			),
		),
	)
}
