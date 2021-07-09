package query

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// audit
	// authz
	// authn
	// process
	// https://stackoverflow.com/a/46021789
	fmt.Fprintf(w, "%s %s %s \n", r.Method, r.URL, r.Proto)
	for k, v := range r.Header {
		fmt.Fprintf(w, "Header field %q, Value %q\n", k, v)
		zap.S().Infof("Header field %q, Value %q", k, v)
	}
	fmt.Fprintf(w, "Host = %q\n", r.Host)
	fmt.Fprintf(w, "RemoteAddr= %q\n", r.RemoteAddr)
}
