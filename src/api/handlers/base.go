package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"server/src/api/controllers"
	"server/src/clients/bcra"
	"server/src/clients/esco"

	"gorm.io/gorm"
)

type Handler struct {
	Controller         controllers.IController
	AccountsController controllers.AccountsControllerI
	ReportsController  controllers.ReportsControllerI
}

func NewHandler(db *gorm.DB, escoClient esco.ESCOServiceClientI, bcraClient bcra.BCRAServiceClientI) (*Handler, error) {
	controller := controllers.NewController(db, escoClient, bcraClient)
	accountsController := controllers.NewAccountsController(escoClient)
	reportsController := controllers.NewReportsController(escoClient, bcraClient, nil)
	return &Handler{Controller: controller, AccountsController: accountsController, ReportsController: reportsController}, nil
}

func (h *Handler) respond(w http.ResponseWriter, _ *http.Request, data interface{}, status int) {
	res, err := json.Marshal(data)
	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(res)
}

func (h *Handler) HandleErrors(w http.ResponseWriter, err error, status int) {
	if err == context.DeadlineExceeded {
		h.respond(w, nil, map[string]string{"error": "Request timed out"}, status)
	} else {
		h.respond(w, nil, map[string]string{"error": err.Error()}, status)
	}
}
