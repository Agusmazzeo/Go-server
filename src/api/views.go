package api

import (
	"net/http"
	handlers "server/src/api/handlers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/config"
	"server/src/database"
	"server/src/repositories"
	"server/src/services"
	redis_utils "server/src/utils/redis"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	"github.com/sirupsen/logrus"
)

type Server struct {
	Router  *chi.Mux
	Handler *handlers.Handler
}

func NewServer(cfg *config.Config, logger *logrus.Logger) (*Server, error) {

	// Initialize Redis
	redis, err := redis_utils.NewRedisHandler(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize Database
	db, err := database.SetupDB(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize Clients
	escoClient, err := esco.NewClient(cfg, redis)
	if err != nil {
		return nil, err
	}
	bcraClient, err := bcra.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize Repositories
	assetCategoryRepository := repositories.NewAssetCategoryRepository(db)
	assetRepository := repositories.NewAssetRepository(db)
	holdingRepository := repositories.NewHoldingRepository(db)
	transactionRepository := repositories.NewTransactionRepository(db)
	syncLogRepository := repositories.NewSyncLogRepository(db)

	// Initialize Services
	escoService := services.NewESCOService(escoClient)
	syncService := services.NewSyncService(
		holdingRepository,
		transactionRepository,
		assetRepository,
		assetCategoryRepository,
		syncLogRepository,
		escoService,
	)
	accountService := services.NewAccountService(holdingRepository, transactionRepository, assetRepository)

	handler, err := handlers.NewHandler(
		logger,
		db,
		escoClient,
		bcraClient,
		escoService,
		syncService,
		accountService,
	)
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
		r.Get("/{ids}", s.Handler.GetReportByIDs)
		r.Get("/{ids}/file", s.Handler.GetReportFileByIDs)
		r.Get("/schedules", s.Handler.GetAllReportSchedules)
		r.Get("/schedule/{id}", s.Handler.GetReportScheduleByID)
		r.Post("/schedule/", s.Handler.CreateReportSchedule)
		r.Put("/schedule/{id}", s.Handler.UpdateReportSchedule)
		r.Delete("/schedule/{id}", s.Handler.DeleteReportSchedule)
	})

	s.Router.Route("/api/accounts", func(r chi.Router) {
		r.Get("/", s.Handler.GetAllAccounts)
		r.Get("/{ids}", s.Handler.GetAccountState)
		r.Post("/sync", s.Handler.SyncAccount)
	})

	s.Router.Route("/api/variables", func(r chi.Router) {
		r.Get("/", s.Handler.GetAllVariables)
		r.Get("/{id}", s.Handler.GetVariableWithValuationByID)
	})

}

// NewHTTPServer creates a new HTTP server with CORS middleware
func NewHTTPServer(cfg *config.Config, logger *logrus.Logger) (*http.Server, error) {
	server, err := NewServer(cfg, logger)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	// Create a custom CORS handler that's more lenient
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Always allow requests without Origin header (server-to-server calls)
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range cfg.CORS.AllowedOrigins {
				if allowedOrigin == "*" {
					allowed = true
					break
				}
				if allowedOrigin == origin {
					allowed = true
					break
				}
				// Handle wildcard patterns like "https://*"
				if len(allowedOrigin) > 2 && allowedOrigin[len(allowedOrigin)-2:] == "/*" {
					prefix := allowedOrigin[:len(allowedOrigin)-1]
					if len(origin) >= len(prefix) && origin[:len(prefix)] == prefix {
						allowed = true
						break
					}
				}
			}

			if allowed {
				// Set CORS headers
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
				if cfg.CORS.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Apply the custom CORS handler
	handler := corsHandler(server.Router)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Service.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      handler,
	}

	return httpServer, nil
}
