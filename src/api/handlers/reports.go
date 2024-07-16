package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"server/src/schemas"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (h *Handler) GetAllReportSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.Controller.GetAllReportSchedules(ctx)

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, nil, 200)
}

func (h *Handler) GetReportScheduleByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get the ID from the URL parameter
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Controller.GetReportScheduleByID(ctx, uint(id))

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, nil, 200)
}

// CreateReportSchedule creates a new report schedule
func (h *Handler) CreateReportSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var reportSchedule schemas.CreateReportScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&reportSchedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.Controller.CreateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.respond(w, r, created, http.StatusCreated)
}

// UpdateReportSchedule updates an existing report schedule
func (h *Handler) UpdateReportSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var reportSchedule schemas.UpdateReportScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&reportSchedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reportSchedule.ID = uint(id)

	updated, err := h.Controller.UpdateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, updated, http.StatusOK)
}

// DeleteReportSchedule deletes an existing report schedule
func (h *Handler) DeleteReportSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Controller.DeleteReportSchedule(ctx, uint(id))
	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, nil, http.StatusNoContent)
}
