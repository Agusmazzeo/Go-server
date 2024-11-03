package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"server/src/api/controllers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Handler struct {
	Logger             *logrus.Logger
	Controller         controllers.IController
	AccountsController controllers.AccountsControllerI
	ReportsController  controllers.ReportsControllerI
}

func NewHandler(logger *logrus.Logger, db *gorm.DB, escoClient esco.ESCOServiceClientI, bcraClient bcra.BCRAServiceClientI) (*Handler, error) {
	controller := controllers.NewController(db, escoClient, bcraClient)
	accountsController := controllers.NewAccountsController(escoClient)
	reportsController := controllers.NewReportsController(escoClient, bcraClient, nil)
	return &Handler{Logger: logger, Controller: controller, AccountsController: accountsController, ReportsController: reportsController}, nil
}

func (h *Handler) respond(w http.ResponseWriter, _ *http.Request, data interface{}, status int) {
	res, err := json.Marshal(data)
	if err != nil {
		h.HandleErrors(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(res)
}

func (h *Handler) HandleErrors(w http.ResponseWriter, err error) {
	var httpErr *utils.HTTPError
	if errors.Is(err, context.DeadlineExceeded) {
		h.respond(w, nil, map[string]string{"error": "Request timed out"}, http.StatusGatewayTimeout)
	} else if errors.As(err, &httpErr) {
		h.respond(w, nil, map[string]string{"error": httpErr.Message}, httpErr.Code)
	} else if err != nil {
		h.respond(w, nil, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
	} else {
		h.respond(w, nil, map[string]string{"error": "Unhandled error"}, http.StatusInternalServerError)
	}

}
