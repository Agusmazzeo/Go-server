package controllers

import (
	"server/src/clients/esco"

	"gorm.io/gorm"
)

type Controller struct {
	DB         *gorm.DB
	ESCOClient *esco.ESCOServiceClient
}

func NewController(db *gorm.DB, escoCLient *esco.ESCOServiceClient) *Controller {
	return &Controller{DB: db, ESCOClient: escoCLient}
}
