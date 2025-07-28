package handlers_test

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"server/src/api/handlers"
	"server/src/config"
	"server/src/repositories"
	"server/src/schemas"
	"server/src/services"
	"server/src/utils"
	bcra_test "server/tests/clients/bcra"
	esco_test "server/tests/clients/esco"
	"server/tests/init_test"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

var ts *httptest.Server
var token *schemas.TokenResponse

func TestMain(m *testing.M) {
	cfg, err := config.LoadConfig("../../../settings", os.Getenv("ENV"))
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	logger := utils.NewLogger(logrus.InfoLevel, false, cfg.Logger.File)

	cfg.ExternalClients.ESCO.CategoryMapFile = "../../test_files/clients/esco/denominaciones.csv"

	escoClient, err := esco_test.NewMockClient("../../test_files/clients/esco")
	if err != nil {
		log.Println(err, "Error while creating Mock Esco Client")
		os.Exit(1)
	}
	bcraClient, err := bcra_test.NewMockClient("../../test_files/clients/bcra")
	if err != nil {
		log.Println(err, "Error while creating Mock BCRA Client")
		os.Exit(1)
	}

	db := init_test.SetupTestDB(nil)

	// Note: Individual tests should handle their own cleanup
	// No global truncate here as each test is responsible for its data

	holdingRepository := repositories.NewHoldingRepository(db)
	transactionRepository := repositories.NewTransactionRepository(db)
	assetRepository := repositories.NewAssetRepository(db)
	assetCategoryRepository := repositories.NewAssetCategoryRepository(db)
	syncLogRepository := repositories.NewSyncLogRepository(db)

	escoService := services.NewESCOService(escoClient)
	syncService := services.NewSyncService(
		holdingRepository,
		transactionRepository,
		assetRepository,
		assetCategoryRepository,
		syncLogRepository,
		escoService,
	)
	if err != nil {
		log.Println(err, "Error while starting handler")
		os.Exit(1)
	}

	r := chi.NewRouter()

	// Create account service
	accountService := services.NewAccountService(holdingRepository, transactionRepository, assetRepository)

	h, err := handlers.NewHandler(logger, db, escoClient, bcraClient, escoService, syncService, accountService)
	if err != nil {
		log.Println(err, "Error while starting handler")
		os.Exit(1)
	}

	r.Route("/api/reports", func(r chi.Router) {
		r.Get("/{id}", h.GetReportFile)
	})

	r.Route("/api/accounts", func(r chi.Router) {
		r.Get("/", h.GetAllAccounts)
		r.Get("/{id}", h.GetAccountState)
	})

	r.Route("/api/variables", func(r chi.Router) {
		r.Get("/", h.GetAllVariables)
		r.Get("/{id}", h.GetVariableWithValuationByID)
	})

	ts = httptest.NewServer(r)
	defer ts.Close()

	token, err = h.Controller.PostToken(context.Background(), "user", "pass")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		os.Exit(1)
	}

	os.Exit(m.Run())
}
