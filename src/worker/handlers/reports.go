package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) LoadAllReportSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := h.Controller.LoadAllReportSchedule(ctx)

	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, nil, 200)
}

func (h *Handler) LoadReportScheduleByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get the ID from the URL parameter
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	err = h.Controller.LoadReportScheduleByID(ctx, uint(id))

	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, nil, 200)
}
