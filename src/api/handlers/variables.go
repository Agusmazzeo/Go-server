package handlers

import (
	"context"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetAllVariables(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var variables []schemas.Variable
	variables, err := h.Controller.GetAllVariables(ctx)

	if err != nil {
		if err == context.DeadlineExceeded {
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, variables, 200)
}

func (h *Handler) GetVariableWithValuationByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	id := chi.URLParam(r, "id")
	var err error

	dateStr := r.URL.Query().Get("date")
	var date time.Time

	startDateStr := r.URL.Query().Get("startDate")
	var startDate time.Time

	endDateStr := r.URL.Query().Get("endDate")
	var endDate time.Time

	var variable *schemas.VariableWithValuationResponse
	if dateStr != "" {
		date, err = time.Parse(utils.ShortDashDateLayout, dateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		date = (date.Add(26 * time.Hour)).In(location)
		variable, err = h.Controller.GetVariableWithValuationByID(ctx, id, date)
	} else if startDateStr != "" && endDateStr != "" {
		startDate, err = time.Parse(utils.ShortDashDateLayout, startDateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		endDate, err = time.Parse(utils.ShortDashDateLayout, endDateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		//Set +26 hours since we use ARG timezone (UTC-3)
		startDate = (startDate.Add(26 * time.Hour)).In(location)
		endDate = (endDate.Add(26 * time.Hour)).In(location)
		variable, err = h.Controller.GetVariableWithValuationDateRangeByID(ctx, id, startDate, endDate)
	}

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, variable, 200)
}
