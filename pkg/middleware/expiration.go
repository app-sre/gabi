package middleware

import (
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/env/user"
)

func Expiration(env *gabi.Env) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if env.UserEnv.IsExpired() {
				l := "The service instance has expired"
				env.Logger.Errorf("%s (expiration date: %s)", l,
					env.UserEnv.Expiration.Format(user.ExpiryDateLayout),
				)
				http.Error(w, l, http.StatusServiceUnavailable)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
