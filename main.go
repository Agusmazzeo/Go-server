package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"server/src/api"
	"server/src/config"
	"server/src/utils"
	"server/src/worker"

	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.LoadConfig("./settings", os.Getenv("ENV"))
	if err != nil {
		panic(err)
	}
	logger := utils.NewLogger(logrus.InfoLevel, false, cfg.Logger.File)
	errC, err := run(cfg, logger)
	if err != nil {
		logger.Error(err, "Error while starting runner")
		return
	}

	if err := <-errC; err != nil {
		logger.Error(err, "Error while running")
	}
}

func run(cfg *config.Config, logger *logrus.Logger) (<-chan error, error) {
	errC := make(chan error, 1)
	var err error

	serviceType := cfg.Service.Type
	port := cfg.Service.Port
	var httpServer *http.Server
	switch serviceType {
	case config.API:
		httpServer, err = api.NewHTTPServer(cfg, logger)
	case config.WORKER:
		httpServer, err = worker.NewHTTPServer(cfg, logger)
	}
	if err != nil {
		return nil, err
	}

	go func() {
		logger.Infoln("Starting server on port", port)

		// "ListenAndServe always returns a non-nil error. After Shutdown or Close, the returned error is
		// ErrServerClosed."
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("An error raised while setting up server", err)
			errC <- err
		}
	}()
	return errC, err
}
