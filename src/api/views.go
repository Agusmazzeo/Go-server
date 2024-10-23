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
	"github.com/rs/cors"
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

	s.Router.Route("/api/reports", func(r chi.Router) {
		r.Get("/{ids}", s.Handler.GetReportFile)
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

// NewHTTPServer creates a new HTTP server with CORS middleware
func NewHTTPServer(cfg *config.Config) (*http.Server, error) {
	server, err := NewServer(cfg)
	if err != nil {
		return nil, err
	}

	// Configure CORS options
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"}, // Allow any localhost config
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            true, // Enable debug for development; remove in production
	})

	// Apply the CORS middleware to your router
	corsHandler := corsMiddleware.Handler(server.Router)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Service.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      corsHandler, // Use the router with CORS enabled
	}

	return httpServer, nil
}
