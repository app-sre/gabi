package handlers

import (
	"encoding/json"
	"net/http"

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

		err := cfg.OverrideDBName(req.DBName)
		if err != nil {
			l := "Unable to open database connection"
			cfg.Logger.Errorf("%s: %s", l, err)
			http.Error(w, l, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"db_name": cfg.GetCurrentDBName()})
	})
}
