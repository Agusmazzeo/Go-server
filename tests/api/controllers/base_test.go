package controllers_test

import (
	"context"
	"log"
	"os"
	"server/src/api/controllers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/config"
	"server/src/repositories"
	"server/src/schemas"
	"server/src/services"
	bcra_test "server/tests/clients/bcra"
	esco_test "server/tests/clients/esco"
	"server/tests/init_test"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var token *schemas.TokenResponse
var escoClient esco.ESCOServiceClientI
var bcraClient bcra.BCRAServiceClientI
var ctrl *controllers.Controller
var accountsController *controllers.AccountsController
var reportsController *controllers.ReportsController
var reportsScheduleController *controllers.ReportScheduleController
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {

	cfg, err := config.LoadConfig("../../../settings", os.Getenv("ENV"))
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	cfg.ExternalClients.ESCO.CategoryMapFile = "../../test_files/utils/denominaciones.csv"

	// escoClient, err = esco.NewClient(cfg)
	escoClient, err = esco_test.NewMockClient("../../test_files/clients/esco")
	if err != nil {
		log.Println(err, "Error while creating ESCO Client")
		os.Exit(1)
	}

	// bcraClient, err = bcra.NewClient(cfg)
	bcraClient, err = bcra_test.NewMockClient("../../test_files/clients/bcra")
	if err != nil {
		log.Println(err, "Error while creating BCRA Client")
		os.Exit(1)
	}

	testDB = init_test.SetupTestDB(nil)

	token, err = escoClient.PostToken(context.Background(), "user", "password")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		// os.Exit(1)
	}

	holdingRepository := repositories.NewHoldingRepository(testDB)
	transactionRepository := repositories.NewTransactionRepository(testDB)
	assetRepository := repositories.NewAssetRepository(testDB)
	assetCategoryRepository := repositories.NewAssetCategoryRepository(testDB)
	syncLogRepository := repositories.NewSyncLogRepository(testDB)

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
	ctrl = controllers.NewController(escoClient, bcraClient)
	accountsController = controllers.NewAccountsController(escoClient, escoService, syncService, accountService)

	// Create report service
	reportService := services.NewReportService()

	// Create report parser service
	reportParserService := services.NewReportParserService()

	reportsController = controllers.NewReportsController(escoClient, bcraClient, reportService, reportParserService, accountService)
	reportsScheduleController = controllers.NewReportScheduleController(testDB)

	os.Exit(m.Run())
}
