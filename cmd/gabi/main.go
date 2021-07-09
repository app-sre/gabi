package main

import (
	"log"

	"go.uber.org/zap"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/cmd"
)

func main() {
	logger, err := zap.NewDevelopment() // NewProduction
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	loggerS := logger.Sugar()
	loggerS.Info("Starting gabi server version " + gabi.Version)

	cmd.Run(loggerS)
}
