package main

import (
	"errors"
	"net/http"
	"server/src/api"
	"server/src/data"

	"log"
)

func main() {
	errC, err := run()
	if err != nil {
		log.Println(err, "Couldn't run")
	}

	if err := <-errC; err != nil {
		log.Println(err, "Error while running")
	}
}

func run() (<-chan error, error) {
	errC := make(chan error, 1)
	dbHandler := data.NewDatabaseHandler()
	server := api.NewServer(dbHandler)
	httpServer := api.NewHTTPServer(server)

	go func() {
		log.Println("Starting server", "port", 8000)

		// "ListenAndServe always returns a non-nil error. After Shutdown or Close, the returned error is
		// ErrServerClosed."
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("An error raised while setting up server", err)
			errC <- err
		}
	}()
	return errC, nil
}
