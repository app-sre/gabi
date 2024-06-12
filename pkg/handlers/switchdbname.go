package handlers

import (
	"encoding/json"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
)

type SwitchDBNameRequest struct {
	DBName string `json:"db_name"`
}

func SwitchDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SwitchDBNameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		cfg.DBEnv.OverrideDBName(req.DBName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"db_name": cfg.DBEnv.GetCurrentDBName()})
	})
}
