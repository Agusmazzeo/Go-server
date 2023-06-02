package api

import (
	"net/http"
	"server/src/data"
)

func (h *Handler) GetEmployees(w http.ResponseWriter, r *http.Request) {
	employees := h.Controller.GetAllEmployees()
	response := map[string][]data.Employees{
		"employees": employees,
	}
	h.respond(w, r, response, 200)
}
