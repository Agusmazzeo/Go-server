package handlers

import (
	"encoding/json"
	"net/http"
	"server/src/api/controllers"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/config"
)

type Handler struct {
	Controller controllers.IController
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	// db, err := database.SetupDB(cfg)
	// if err != nil {
	// 	return nil, err
	// }
	escoClient, err := esco.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	bcraClient, err := bcra.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	controller := controllers.NewController(nil, escoClient, bcraClient)
	return &Handler{Controller: controller}, nil
}

func (s *Handler) respond(w http.ResponseWriter, _ *http.Request, data interface{}, status int) {
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
