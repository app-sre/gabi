package main

import (
	"database/sql"
	"log"

	"go.uber.org/zap"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/cmd"
)

var DB *sql.DB // Replace with dependency injection for logger and db pool

func main() {
	logger, err := zap.NewDevelopment() // NewProduction
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	undo := zap.ReplaceGlobals(logger)
	defer undo()

	zap.S().Info("Starting gabi server version " + gabi.Version)

	cmd.Run()
}
