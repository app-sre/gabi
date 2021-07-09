package handlers

import (
	"fmt"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
	"go.uber.org/zap"
)

func Query(env *gabi.Env) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// https://stackoverflow.com/a/46021789
		env.Logger.Info("testing")
		fmt.Fprintf(w, "%s %s %s \n", r.Method, r.URL, r.Proto)
		for k, v := range r.Header {
			fmt.Fprintf(w, "Header field %q, Value %q\n", k, v)
			zap.S().Infof("Header field %q, Value %q", k, v)
		}
		fmt.Fprintf(w, "Host = %q\n", r.Host)
		fmt.Fprintf(w, "RemoteAddr= %q\n", r.RemoteAddr)
	}
}
