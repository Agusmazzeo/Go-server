package controllers_test

import (
	"server/src/api/controllers"
	"server/tests/init_test"
	"testing"

	"github.com/go-logr/logr"
)

func TestGetAllAccounts(t *testing.T) {
	db, cleanup := init_test.SetUpTestDatabase(t, &logr.Logger{})
	defer cleanup()

	_ = controllers.Controller{DB: db}

}
