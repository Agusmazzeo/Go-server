package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"server/src/config"
	"server/src/database"
	"server/src/utils"
	"server/src/worker/controllers"
)

type Handler struct {
	Controller *controllers.Controller
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	db, err := database.SetupDB(cfg)
	if err != nil {
		return nil, err
	}
	controller := controllers.NewController(db)
	return &Handler{Controller: controller}, nil
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
