package worker

import (
	"net/http"
	"server/src/config"
	handlers "server/src/worker/handlers"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	Router  *chi.Mux
	Handler *handlers.Handler
}

func NewServer(cfg *config.Config) (*Server, error) {
	handler, err := handlers.NewHandler(cfg)
	if err != nil {
		return nil, err
	}
	server := &Server{
		Router:  chi.NewRouter(),
		Handler: handler,
	}
	server.InitRoutes()
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) InitRoutes() {
	s.Router.Get("/alive", handlers.Healthcheck)
	s.Router.Route("/api/report", func(r chi.Router) {
		r.Post("/all", s.Handler.LoadAllReportSchedules)
		r.Post("/{id}", s.Handler.LoadReportScheduleByID)
	})
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
