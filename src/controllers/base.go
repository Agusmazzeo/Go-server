package controllers

import "server/src/data"

type Controller struct {
	DbHandler *data.DatabaseHandler
}

func NewController(dbHandler *data.DatabaseHandler) *Controller {
	return &Controller{DbHandler: dbHandler}
}
