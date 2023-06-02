package api

import (
	"net/http"
	"server/src/data"
)

func (h *Handler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	customers := h.Controller.GetAllCustomers()
	response := map[string][]data.Customers{
		"customers": customers,
	}
	h.respond(w, r, response, 200)
}
