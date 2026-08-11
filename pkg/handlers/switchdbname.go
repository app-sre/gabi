package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
	"github.com/app-sre/gabi/pkg/env/db"
	"github.com/app-sre/gabi/pkg/middleware"
	"github.com/app-sre/gabi/pkg/models"
)

func SwitchDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := r.Context().Value(middleware.ContextKeyUser).(string)
		if user == "" {
			http.Error(w, "Request cannot be authorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, gabi.MaxRequestBodyBytes)

		var req models.SwitchDBNameRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		// Reject trailing junk after the first JSON value.
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := db.ValidateDBName(req.DBName); err != nil {
			cfg.Logger.Errorf("Rejected database switch: %s", err)
			http.Error(w, "Invalid database name", http.StatusBadRequest)
			return
		}

		// Audit before mutation (fail closed), matching /query behavior.
		query := &audit.QueryData{
			Query:     fmt.Sprintf("SWITCH DATABASE %s", req.DBName),
			User:      user,
			Timestamp: time.Now().Unix(),
		}
		_ = cfg.LoggerAudit.Write(r.Context(), query)
		if err := cfg.SplunkAudit.Write(r.Context(), query); err != nil {
			cfg.Logger.Errorf("Unable to send audit to Splunk: %s", err)
			http.Error(w, "An internal error has occurred", http.StatusInternalServerError)
			return
		}

		if err := cfg.OverrideDBName(req.DBName); err != nil {
			l := "Unable to open database connection"
			cfg.Logger.Errorf("%s: %s", l, err)
			http.Error(w, l, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"db_name": cfg.GetCurrentDBName()})
	})
}
