package middleware

import (
	"errors"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
)

func Recovery(env *gabi.Env) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if ok && errors.Is(err, http.ErrAbortHandler) {
						panic(err)
					}

					env.Logger.Errorf("Recovered from an error: %s", r)
					http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}
