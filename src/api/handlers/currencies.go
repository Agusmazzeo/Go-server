package handlers

import (
	"context"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetAllCurrencies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var currencies []schemas.Currency
	currencies, err := h.Controller.GetAllCurrencies(ctx)

	if err != nil {
		if err == context.DeadlineExceeded {
			h.HandleErrors(w, err, http.StatusGatewayTimeout)
		} else {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, currencies, 200)
}

func (h *Handler) GetCurrencyWithValuationByID(w http.ResponseWriter, r *http.Request) {
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

	var currency *schemas.CurrencyWithValuationResponse
	if dateStr != "" {
		date, err = time.Parse(utils.ShortDashDateLayout, dateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		date = (date.Add(26 * time.Hour)).In(location)
		currency, err = h.Controller.GetCurrencyWithValuationByID(ctx, id, date)
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
		currency, err = h.Controller.GetCurrencyWithValuationDateRangeByID(ctx, id, startDate, endDate)
	}

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, currency, 200)
}
