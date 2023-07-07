package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/models"
)

func Audit(cfg *gabi.Config) Middleware {
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

			base64DecodeQuery := false
			if s := r.URL.Query().Get("base64_query"); s != "" {
				if ok, err := strconv.ParseBool(s); err == nil && ok {
					base64DecodeQuery = true
				}
			}

			if ctxUser := ctx.Value(ContextKeyUser); ctxUser != nil {
				if s, ok := ctxUser.(string); ok {
					user = s
				}
			} else {
				user = r.Header.Get(forwardedUserHeader)
			}
			if user == "" {
				l := fmt.Sprintf("Request without required header: %s", forwardedUserHeader)
				http.Error(w, l, http.StatusBadRequest)
				return
			}

			if _, err := io.Copy(&b, r.Body); err != nil {
				cfg.Logger.Errorf("Unable to copy request body: %s", err)
				http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				return
			}
			_ = r.Body.Close()

			r.Body = io.NopCloser(bytes.NewReader(b.Bytes()))

			err := json.Unmarshal(b.Bytes(), &request)
			if err != nil {
				cfg.Logger.Debugf("Unable to unmarshal request body: %s", err)
				h.ServeHTTP(w, r)
				return
			}

			if base64DecodeQuery {
				bytes, err := cfg.Encoder.DecodeString(request.Query)
				if err != nil {
					l := "Unable to decode Base64-encoded query"
					cfg.Logger.Errorf("%s: %s", l, err)
					http.Error(w, l, http.StatusBadRequest)
					return
				}
				request.Query = string(bytes)
			}

			query := &audit.QueryData{
				Query:     request.Query,
				User:      user,
				Timestamp: now.Unix(),
			}
			_ = cfg.LoggerAudit.Write(ctx, query)

			if err := cfg.SplunkAudit.Write(ctx, query); err != nil {
				cfg.Logger.Errorf("Unable to send audit to Splunk: %s", err)
				http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, ContextKeyQuery, request.Query)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
