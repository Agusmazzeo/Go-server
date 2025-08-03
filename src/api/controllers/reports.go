package controllers

import (
	"context"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/services"
	"time"

	"github.com/xuri/excelize/v2"
)

type ReportsControllerI interface {
	GetReport(ctx context.Context, token string, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)
	GenerateXLSXReportFromClientIDs(ctx context.Context, token string, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error)
	GeneratePDFReportFromClientIDs(ctx context.Context, token string, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) ([]byte, error)
}

type ReportsController struct {
	ESCOClient          esco.ESCOServiceClientI
	BCRAClient          bcra.BCRAServiceClientI
	ReportService       services.ReportServiceI
	ReportParserService services.ReportParserServiceI
	AccountService      services.AccountServiceI
	SyncService         services.SyncServiceI
}

func NewReportsController(
	escoClient esco.ESCOServiceClientI,
	bcraClient bcra.BCRAServiceClientI,
	reportService services.ReportServiceI,
	reportParserService services.ReportParserServiceI,
	accountService services.AccountServiceI,
	syncService services.SyncServiceI,
) *ReportsController {
	return &ReportsController{
		ESCOClient:          escoClient,
		BCRAClient:          bcraClient,
		ReportService:       reportService,
		ReportParserService: reportParserService,
		AccountService:      accountService,
		SyncService:         syncService,
	}
}

func (rc *ReportsController) GetReport(
	ctx context.Context,
	token string,
	clientIDs []string,
	variablesWithValuations map[string]*schemas.VariableWithValuationResponse,
	startDate, endDate time.Time,
	interval time.Duration,
) (*schemas.AccountsReports, error) {
	// Sync data from client IDs
	for _, id := range clientIDs {
		err := rc.SyncService.SyncDataFromAccount(ctx, token, id, startDate, endDate)
		if err != nil {
			return nil, err
		}
	}

	// Build account state from client ID using existing AccountService
	accountStateByCategory, err := rc.AccountService.GetMultiAccountStateByCategory(ctx, clientIDs, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate report from account state
	accountReports, err := rc.ReportService.GenerateReport(ctx, accountStateByCategory, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	accountReports.ReferenceVariables = &variablesWithValuations
	return accountReports, nil
}

func (rc *ReportsController) GenerateXLSXReportFromClientIDs(ctx context.Context, token string, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error) {
	// Get the report data
	accountsReport, err := rc.GetReport(ctx, token, clientIDs, variablesWithValuations, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate dataframes
	dataframes, err := rc.ReportService.GenerateReportDataframes(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate XLSX file
	return rc.ReportService.GenerateXLSXReport(ctx, dataframes)
}

func (rc *ReportsController) GeneratePDFReportFromClientIDs(ctx context.Context, token string, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) ([]byte, error) {
	// Get the report data
	accountsReport, err := rc.GetReport(ctx, token, clientIDs, variablesWithValuations, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate dataframes
	dataframes, err := rc.ReportService.GenerateReportDataframes(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate PDF file
	return rc.ReportParserService.ParseAccountsReportToPDF(ctx, dataframes)
}
