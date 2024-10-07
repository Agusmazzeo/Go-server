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
	"testing"
)

var token *schemas.TokenResponse
var escoClient *esco.ESCOServiceClient
var bcraClient *bcra.BCRAServiceClient
var ctrl *controllers.Controller

func TestMain(m *testing.M) {

	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	cfg.ExternalClients.ESCO.CategoryMapFile = "../../test_files/utils/denominaciones.csv"

	escoClient, err = esco.NewClient(cfg)
	if err != nil {
		log.Println(err, "Error while creating ESCO Client")
		os.Exit(1)
	}

	bcraClient, err = bcra.NewClient(cfg)
	if err != nil {
		log.Println(err, "Error while creating BCRA Client")
		os.Exit(1)
	}

	token, err = escoClient.PostToken(context.Background(), "icastagno", "Messiusa24!")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		// os.Exit(1)
	}

	ctrl = controllers.NewController(nil, escoClient, bcraClient)

	os.Exit(m.Run())
}
