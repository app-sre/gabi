package middleware

import (
	"context"
	"fmt"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
)

func Authorization(env *gabi.Env) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user := r.Header.Get(forwardedUserHeader)
			if user == "" {
				l := fmt.Sprintf("Request without required header: %s", forwardedUserHeader)
				http.Error(w, l, http.StatusBadRequest)
				return
			}

			if len(env.UserEnv.Users) == 0 {
				http.Error(w, "Request cannot be authorized", http.StatusUnauthorized)
				return
			}
			for _, u := range env.UserEnv.Users {
				if user == u {
					ctx = context.WithValue(ctx, contextUserKey, u)
					h.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			l := "User does not have required permissions"
			env.Logger.Errorf("%s: %s", l, user)
			http.Error(w, l, http.StatusForbidden)
		})
	}
}
