package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/models"
)

func GetDBName(cfg *gabi.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbName := cfg.DBEnv.GetCurrentDBName()
		defaultDBName := os.Getenv("DB_NAME")

		response := models.DBNameResponse{DBName: dbName}

		if dbName != defaultDBName {
			warning := "Current database differs from the default"
			cfg.Logger.Warnf(warning)
			response.Warnings = append(response.Warnings, warning)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}
