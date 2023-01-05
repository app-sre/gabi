package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/models"
)

func Audit(env *gabi.Env) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			now := time.Now()

			var (
				b       bytes.Buffer
				request models.QueryRequest
				user    string
			)

			if s := r.Header.Get(contentLengthHeader); s == "" {
				l := fmt.Sprintf("Request without required header: %s", contentLengthHeader)
				http.Error(w, l, http.StatusBadRequest)
				return
			}

			ctxUser := ctx.Value(contextUserKey)
			if ctxUser != nil {
				user = ctxUser.(string)
			} else {
				user = r.Header.Get(forwardedUserHeader)
			}
			if user == "" {
				l := fmt.Sprintf("Request without required header: %s", forwardedUserHeader)
				http.Error(w, l, http.StatusBadRequest)
				return
			}

			if _, err := io.Copy(&b, r.Body); err != nil {
				env.Logger.Errorf("Unable to copy request body: %s", err)
				http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				return
			}
			_ = r.Body.Close()

			r.Body = io.NopCloser(bytes.NewReader(b.Bytes()))

			err := json.Unmarshal(b.Bytes(), &request)
			if err != nil {
				env.Logger.Debugf("Unable to unmarshal request body: %s", err)
				h.ServeHTTP(w, r)
				return
			}

			query := &audit.QueryData{
				Query:     request.Query,
				User:      user,
				Timestamp: now.Unix(),
			}
			_ = env.LoggerAudit.Write(query)

			if err := env.SplunkAudit.Write(query); err != nil {
				env.Logger.Errorf("Unable to send audit to Splunk: %s", err)
				http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
