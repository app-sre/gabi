package middleware

import (
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/user"
)

func Expiration(cfg *gabi.Config) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.UserEnv.IsExpired() {
				l := "The service instance has expired"
				cfg.Logger.Errorf("%s (expiration date: %s)", l,
					cfg.UserEnv.Expiration.Format(user.ExpiryDateLayout),
				)
				http.Error(w, l, http.StatusServiceUnavailable)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
