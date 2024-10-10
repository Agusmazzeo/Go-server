package api

import (
	"net/http"
	handlers "server/src/api/handlers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/config"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
)

type Server struct {
	Router  *chi.Mux
	Handler *handlers.Handler
}

func NewServer(cfg *config.Config) (*Server, error) {
	// db, err := database.SetupDB(cfg)
	// if err != nil {
	// 	return nil, err
	// }
	escoClient, err := esco.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	bcraClient, err := bcra.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	handler, err := handlers.NewHandler(nil, escoClient, bcraClient)
	if err != nil {
		return nil, err
	}
	r := chi.NewRouter()
	var tokenAuth *jwtauth.JWTAuth = jwtauth.New("HS256", []byte("secret"), nil)
	r.Use(jwtauth.Verifier(tokenAuth))
	r.Use(jwtauth.Authenticator)
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
	s.Router.Get("/", handlers.Healthcheck)
	s.Router.Get("/api/alive", handlers.Healthcheck)

	s.Router.Post("/api/token", s.Handler.PostToken)

	s.Router.Route("/api/report", func(r chi.Router) {
		r.Get("/schedules", s.Handler.GetAllReportSchedules)
		r.Get("/schedule/{id}", s.Handler.GetReportScheduleByID)
		r.Post("/schedule/", s.Handler.CreateReportSchedule)
		r.Put("/schedule/{id}", s.Handler.UpdateReportSchedule)
		r.Delete("/schedule/{id}", s.Handler.DeleteReportSchedule)
	})

	s.Router.Route("/api/accounts", func(r chi.Router) {
		r.Get("/", s.Handler.GetAllAccounts)
		r.Get("/{id}", s.Handler.GetAccountState)
	})

	s.Router.Route("/api/currencies", func(r chi.Router) {
		r.Get("/", s.Handler.GetAllCurrencies)
		r.Get("/{id}", s.Handler.GetCurrencyWithValuationByID)
	})

	s.Router.Route("/api/variables", func(r chi.Router) {
		r.Get("/", s.Handler.GetAllVariables)
		r.Get("/{id}", s.Handler.GetVariableWithValuationByID)
	})

}

func NewHTTPServer(cfg *config.Config) (*http.Server, error) {
	server, err := NewServer(cfg)
	if err != nil {
		return nil, err
	}
	httpServer := &http.Server{
		Addr:         ":" + cfg.Service.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      server,
	}
	return httpServer, nil
}
