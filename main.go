package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"server/src/api"
	"server/src/config"
	"server/src/worker"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.LoadConfig()
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

	// Setup GORM
	dsn := "host=" + cfg.Databases.SQL.Host + " user=" + cfg.Databases.SQL.Username + " password=" + cfg.Databases.SQL.Password + " dbname=" + cfg.Databases.SQL.Database + " port=" + cfg.Databases.SQL.Port + " sslmode=disable"
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	serviceType := os.Getenv("SERVICE_TYPE")
	var httpServer *http.Server
	if serviceType == "API" {
		// Initialize the API server with GORM DB
		server := api.NewServer(db)
		httpServer = api.NewHTTPServer(server)
	} else {
		// Initialize the Worker server with GORM DB
		server := worker.NewServer(db)
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
