package middleware

import (
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
)

func Recovery(env *gabi.Env) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if err == http.ErrAbortHandler {
						panic(err)
					}
					env.Logger.Errorf("Recovered from an error: %s", err)
					http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}
