package api

import (
	"encoding/json"
	"net/http"
	"server/src/controllers"
	"server/src/data"
)

type Handler struct {
	Controller controllers.Controller
}

func NewHandler(dbHandler *data.DatabaseHandler) *Handler {
	controller := controllers.NewController(dbHandler)
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
