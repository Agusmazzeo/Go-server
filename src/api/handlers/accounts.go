package handlers

import (
	"context"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	filter := r.URL.Query().Get("filter")
	accounts, err := h.Controller.GetAllAccounts(ctx, filter)

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, accounts, 200)
}

func (h *Handler) GetAccountState(w http.ResponseWriter, r *http.Request) {
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

	var accountState *schemas.AccountState
	if dateStr != "" {
		date, err = time.Parse(utils.ShortDashDateLayout, dateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		date = (date.Add(26 * time.Hour)).In(location)
		accountState, err = h.Controller.GetAccountState(ctx, id, date)
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
		accountState, err = h.Controller.GetAccountStateDateRange(ctx, id, startDate, endDate)
	}

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, accountState, 200)
}

func (h *Handler) HandleErrors(w http.ResponseWriter, err error, status int) {
	if err == context.DeadlineExceeded {
		http.Error(w, "Request timed out", status)
	} else {
		http.Error(w, err.Error(), status)
	}
}
