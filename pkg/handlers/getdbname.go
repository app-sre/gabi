package handlers

import (
	"encoding/json"
	"net/http"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/models"
)

func GetCurrentDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbName := cfg.GetCurrentDBName()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.DBNameResponse{DBName: dbName})
	})
}
