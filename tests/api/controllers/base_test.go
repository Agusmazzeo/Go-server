package controllers_test

import (
	"context"
	"log"
	"os"
	"server/src/api/controllers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/config"
	"server/src/schemas"
	bcra_test "server/tests/clients/bcra"
	esco_test "server/tests/clients/esco"
	"testing"
)

var token *schemas.TokenResponse
var escoClient esco.ESCOServiceClientI
var bcraClient bcra.BCRAServiceClientI
var ctrl *controllers.Controller

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

	token, err = escoClient.PostToken(context.Background(), "user", "password")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		// os.Exit(1)
	}

	ctrl = controllers.NewController(nil, escoClient, bcraClient)

	os.Exit(m.Run())
}
