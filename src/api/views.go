package api

import (
	"net/http"
	handlers "server/src/api/handlers"
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

	s.Router.Route("/api/report", func(r chi.Router) {
		r.Get("/schedules", s.Handler.GetAllReportSchedules)
		r.Get("/schedule/{id}", s.Handler.GetReportScheduleByID)
		r.Post("/schedule/", s.Handler.CreateReportSchedule)
		r.Put("/schedule/{id}", s.Handler.UpdateReportSchedule)
		r.Delete("/schedule/{id}", s.Handler.DeleteReportSchedule)
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
