package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"server/src/schemas"
	"time"
)

func (h *Handler) PostToken(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var tokenRequestCreds = new(schemas.TokenRequest)

	err := json.NewDecoder(r.Body).Decode(tokenRequestCreds)
	if err != nil {
		h.HandleErrors(w, err, http.StatusBadRequest)
		return
	}

	tokenResponse, err := h.Controller.PostToken(ctx, tokenRequestCreds.Username, tokenRequestCreds.Password)
	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}

	h.respond(w, r, tokenResponse, 200)
}
