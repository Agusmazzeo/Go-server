package handlers

import (
	"encoding/json"
	"net/http"
	"server/src/config"
	"server/src/database"
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
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(res)
}
