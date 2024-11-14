package controllers_test

import (
	"context"
	"fmt"
	"os"
	"server/src/schemas"
	"server/src/utils"
	"testing"
	"time"
)

func TestGetAllAccounts(t *testing.T) {

	accounts, err := accountsController.GetAllAccounts(context.Background(), token.AccessToken, "DIAGNOSTICO VETERINARIO")
	if err != nil {
		t.Error(err)
	}

	if len(accounts) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}

func TestGetAccountByID(t *testing.T) {

	account, err := accountsController.GetAccountByID(context.Background(), token.AccessToken, "11170") // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if account == nil {
		t.Errorf("expected account to be returned")
	}
}

func TestGetAccountState(t *testing.T) {

	// Use a valid account ID and date here
	date := time.Now()
	accountState, err := accountsController.GetAccountState(context.Background(), token.AccessToken, "11170", date)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state")
	}
}

func TestGetAccountStateDateRange(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	interval, _ := utils.ParseTimeInterval("1w:0d")
	accountState, err := accountsController.GetAccountStateDateRange(context.Background(), token.AccessToken, "11170", startDate, endDate, interval.ToDuration())
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state for the date range")
	}
}

func TestGetBoletosDateRange(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 6, 25, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	accountState, err := accountsController.GetBoletosDateRange(context.Background(), token.AccessToken, "11170", startDate, endDate)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state for the date range")
	}
}

func TestGetLiquidacionesDateRange(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 6, 25, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	accountState, err := accountsController.GetLiquidacionesDateRange(context.Background(), token.AccessToken, "11170", startDate, endDate)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state for the date range")
	}
}

func TestCollapseAndGroupAccountsStates(t *testing.T) {
	const inputFile = "../../test_files/controllers/accounts/accounts_states.json"
	const outputDir = "../../test_files/controllers/accounts"
	var outputFile = fmt.Sprintf("%s/collapsed_account_states.json", outputDir)

	// Load the input data
	var accountsStates []*schemas.AccountState
	err := utils.LoadStructFromJSONFile(inputFile, &accountsStates)
	if err != nil {
		t.Fatalf("Failed to load input data: %v", err)
	}

	for i := 0; i < 5; i++ {
		// Generate the report
		collapsedAccounts := accountsController.CollapseAndGroupAccountsStates(accountsStates)
		if err != nil {
			t.Fatalf("Failed to generate account report: %v", err)
		}

		// Check if the output file already exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			// File does not exist, so create it
			err = utils.SaveStructToJSONFile(collapsedAccounts, outputFile)
			if err != nil {
				t.Fatalf("Failed to save output report: %v", err)
			}
			t.Logf("Output file created: %s", outputFile)
		} else {
			// File exists, load the saved report and compare
			var savedReport schemas.AccountStateByCategory
			err = utils.LoadStructFromJSONFile(outputFile, &savedReport)
			if err != nil {
				t.Fatalf("Failed to load saved report: %v", err)
			}

			// Compare the newly generated report with the saved one
			if !compareVouchersByCategory(t, *collapsedAccounts.VouchersByCategory, *savedReport.VouchersByCategory) {
				t.Errorf("Generated report does not match the saved report on iteration %d", i+1)
				outputFile := fmt.Sprintf("collapsed_account_states_%d.json", i+1)
				err = utils.SaveStructToJSONFile(collapsedAccounts, fmt.Sprintf("%s/%s", outputDir, outputFile))
				if err != nil {
					t.Fatalf("Failed to save output report: %v", err)
				}
			} else {
				t.Logf("Generated report matches the saved report on iteration %d", i+1)
			}
		}
	}
}
