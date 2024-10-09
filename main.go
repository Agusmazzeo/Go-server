package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"server/src/api"
	"server/src/config"
	"server/src/worker"
)

func main() {
	cfg, err := config.LoadConfig("./settings", os.Getenv("ENV"))
	if err != nil {
		log.Println(err, "Error while loading config")
		return
	}
	errC, err := run(cfg)
	if err != nil {
		log.Println(err, "Couldn't run")
		return
	}

	if err := <-errC; err != nil {
		log.Println(err, "Error while running")
	}
}

func run(cfg *config.Config) (<-chan error, error) {
	errC := make(chan error, 1)
	var err error

	serviceType := cfg.Service.Type
	port := cfg.Service.Port
	var httpServer *http.Server
	switch serviceType {
	case config.API:
		httpServer, err = api.NewHTTPServer(cfg)
	case config.WORKER:
		httpServer, err = worker.NewHTTPServer(cfg)
	}
	if err != nil {
		return nil, err
	}

	go func() {
		log.Println("Starting server on port", port)

		// "ListenAndServe always returns a non-nil error. After Shutdown or Close, the returned error is
		// ErrServerClosed."
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("An error raised while setting up server", err)
			errC <- err
		}
	}()
	return errC, err
}
