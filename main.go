package main

import (
	"errors"
	"log"
	"net/http"
	"server/src/api"
	"server/src/config"
	"server/src/worker"
)

func main() {
	cfg, err := config.LoadConfig("./settings")
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

	serviceType := cfg.Service.Type
	var httpServer *http.Server
	if serviceType == "API" {
		// Initialize the API server with GORM DB
		server, err := api.NewServer(cfg)
		if err != nil {
			return nil, err
		}
		httpServer = api.NewHTTPServer(server)
	} else {
		// Initialize the Worker server with GORM DB
		server, err := worker.NewServer(cfg)
		if err != nil {
			return nil, err
		}
		httpServer = worker.NewHTTPServer(server)
	}

	go func() {
		log.Println("Starting server on port", 8000)

		// "ListenAndServe always returns a non-nil error. After Shutdown or Close, the returned error is
		// ErrServerClosed."
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("An error raised while setting up server", err)
			errC <- err
		}
	}()
	return errC, nil
}
