package handlers

import (
	"context"
	"net/http"
	"time"
)

func (h *Handler) GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	filter := r.URL.Query().Get("filter")
	accounts, err := h.Controller.GetAllAccounts(ctx, filter)

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, accounts, 200)
}
