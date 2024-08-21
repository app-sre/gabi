package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/models"
)

func SwitchDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.SwitchDBNameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		oldDBName := cfg.DBEnv.GetCurrentDBName()
		cfg.DBEnv.OverrideDBName(req.DBName)
		newDBName := cfg.DBEnv.GetCurrentDBName()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := cfg.DB.PingContext(ctx); err != nil {
			cfg.Logger.Errorf("Failed to ping new database %s, falling back to %s: %s", newDBName, oldDBName, err)
			cfg.DBEnv.OverrideDBName(oldDBName)
			newDBName = oldDBName
		} else {
			cfg.Logger.Infof("Database name switched from %s to %s", oldDBName, newDBName)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"db_name": newDBName}); err != nil {
			cfg.Logger.Errorf("Failed to encode response: %s", err)
		}
	})
}
