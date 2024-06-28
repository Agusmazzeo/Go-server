package handlers

import (
	"encoding/json"
	"net/http"
	"server/src/worker/controllers"

	"gorm.io/gorm"
)

type Handler struct {
	Controller controllers.Controller
}

func NewHandler(db *gorm.DB) *Handler {
	controller := controllers.NewController(db)
	return &Handler{Controller: *controller}
}

func (s *Handler) respond(w http.ResponseWriter, r *http.Request, data interface{}, status int) {
	res, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(res)
}
