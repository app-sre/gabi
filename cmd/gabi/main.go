package main

import (
	"log"

	"github.com/app-sre/gabi/pkg/cmd"
	"go.uber.org/zap"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Unable to initialize Zap logger: %s", err)
	}
	defer func() { _ = l.Sync() }()

	logger := l.Sugar()
	if err := cmd.Run(logger); err != nil {
		logger.Fatalf("Unable to start GABI: %s", err)
	}
}
