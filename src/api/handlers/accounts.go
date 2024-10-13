package handlers

import (
	"context"
	"fmt"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
)

func (h *Handler) GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token := jwtauth.TokenFromHeader(r)

	filter := r.URL.Query().Get("filter")
	accounts, err := h.AccountsController.GetAllAccounts(ctx, token, filter)

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, accounts, 200)
}

func (h *Handler) GetAccountState(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, fmt.Errorf("empty token detected"), http.StatusUnauthorized)
	}

	id := chi.URLParam(r, "id")
	var err error

	dateStr := r.URL.Query().Get("date")
	var date time.Time

	startDateStr := r.URL.Query().Get("startDate")
	var startDate time.Time

	endDateStr := r.URL.Query().Get("endDate")
	var endDate time.Time

	intervalStr := r.URL.Query().Get("interval")
	if intervalStr == "" {
		// Set interval per day as default
		intervalStr = "0m:0w:1d"
	}
	interval, err := utils.ParseTimeInterval(intervalStr)
	if err != nil {
		h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		return
	}

	var accountState *schemas.AccountState
	if dateStr != "" {
		date, err = time.Parse(utils.ShortDashDateLayout, dateStr)
		if err != nil {
			h.HandleErrors(w, err, http.StatusUnprocessableEntity)
		}
		date = (date.Add(26 * time.Hour)).In(location)
		accountState, err = h.AccountsController.GetAccountState(ctx, token, id, date)
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
		accountState, err = h.AccountsController.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval.ToDuration())
	}

	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
	}

	h.respond(w, r, accountState, 200)
}

func (h *Handler) HandleErrors(w http.ResponseWriter, err error, status int) {
	if err == context.DeadlineExceeded {
		h.respond(w, nil, map[string]string{"error": "Request timed out"}, status)
	} else {
		h.respond(w, nil, map[string]string{"error": err.Error()}, status)
	}
}
