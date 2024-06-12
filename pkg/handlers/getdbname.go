package handlers

import (
	"encoding/json"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
)

func GetCurrentDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbName := cfg.DBEnv.GetCurrentDBName()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"db_name": dbName})
	})
}
