package worker

import (
	"net/http"
	handlers "server/src/worker/handlers"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Server struct {
	Router  *chi.Mux
	Handler handlers.Handler
}

func NewServer(db *gorm.DB) *Server {
	server := &Server{
		Router:  chi.NewRouter(),
		Handler: *handlers.NewHandler(db),
	}
	server.InitRoutes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) InitRoutes() {
	s.Router.Get("/alive", handlers.Healthcheck)
	s.Router.Post("/api/report", handlers.ReportsHandler)

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
