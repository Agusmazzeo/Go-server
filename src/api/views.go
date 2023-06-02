package api

import (
	"fmt"
	"net/http"
	handlers "server/src/api/handlers"
	"server/src/data"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	Router  *chi.Mux
	Handler handlers.Handler
}

func NewServer(dbHandler *data.DatabaseHandler) *Server {
	server := &Server{
		Router:  chi.NewRouter(),
		Handler: *handlers.NewHandler(dbHandler),
	}
	server.InitRoutes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) InitRoutes() {
	s.Router.Get("/alive", handlers.Healthcheck)

	s.Router.Get("/customers", s.Handler.GetCustomers)
	s.Router.Get("/employees", s.Handler.GetEmployees)
}

func InitViews() {
	http.HandleFunc("/alive", handlers.Healthcheck)

	fmt.Println("Starting server on port 8000")
	http.ListenAndServe(":8000", nil)
}

func NewHTTPServer(server *Server) *http.Server {
	httpServer := &http.Server{
		Addr:         ":" + "8000",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      server,
	}
	return httpServer
}
