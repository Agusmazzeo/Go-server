package controllers_test

import (
	"context"
	"server/src/api/controllers"
	"server/src/clients/esco"
	"server/src/config"
	"server/tests/init_test"
	"testing"

	"github.com/go-logr/logr"
)

func TestGetAllAccounts(t *testing.T) {
	db, cleanup := init_test.SetUpTestDatabase(t, &logr.Logger{})
	defer cleanup()

	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		t.Error(err)
	}
	escoClient := esco.NewClient(cfg)

	ctrl := controllers.NewController(db, escoClient)

	accounts, err := ctrl.GetAllAccounts(context.Background(), "DIAGNOSTICO VETERINARIO")
	if err != nil {
		t.Error(err)
	}

	if len(accounts) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}
