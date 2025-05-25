package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"strings"
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
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, accounts, 200)
}

func (h *Handler) GetAccountState(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusBadRequest, "empty token detected"))
		return
	}
	var err error
	idsStr := chi.URLParam(r, "ids")
	// Split the comma-separated ids into a slice
	ids := strings.Split(idsStr, ",")

	if len(ids) == 0 {
		http.Error(w, "Missing id URL parameter", http.StatusBadRequest)
		return
	}

	dateStr := r.URL.Query().Get("date")
	var date time.Time

	startDateStr := r.URL.Query().Get("startDate")
	var startDate time.Time

	endDateStr := r.URL.Query().Get("endDate")
	var endDate time.Time

	intervalStr := r.URL.Query().Get("interval")
	if intervalStr == "" {
		// Set interval per day as default
		intervalStr = "0w:1d"
	}
	interval, err := utils.ParseTimeInterval(intervalStr)
	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	var accountState *schemas.AccountStateByCategory
	if dateStr != "" {
		date, err = time.Parse(utils.ShortDashDateLayout, dateStr)
		if err != nil {
			h.HandleErrors(w, err)
			return
		}
		date = (date.Add(26 * time.Hour)).In(location)
		accountState, err = h.AccountsController.GetMultiAccountStateByCategoryDateRange(ctx, token, ids, date, date, interval.ToDuration())
	} else if startDateStr != "" && endDateStr != "" {
		startDate, err = time.Parse(utils.ShortDashDateLayout, startDateStr)
		if err != nil {
			h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
			return
		}
		endDate, err = time.Parse(utils.ShortDashDateLayout, endDateStr)
		if err != nil {
			h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
			return
		}
		//Set +26 hours since we use ARG timezone (UTC-3)
		startDate = (startDate.Add(26 * time.Hour)).In(location)
		endDate = (endDate.Add(26 * time.Hour)).In(location)
		accountState, err = h.AccountsController.GetMultiAccountStateByCategoryDateRange(ctx, token, ids, startDate, endDate, interval.ToDuration())
	}

	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, accountState, 200)
}

// SyncAccount handles the POST request to sync account data
func (h *Handler) SyncAccount(w http.ResponseWriter, r *http.Request) {
	var syncRequest schemas.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&syncRequest); err != nil {
		h.HandleErrors(w, err)
		return
	}

	// Validate request
	if syncRequest.AccountID == "" {
		h.HandleErrors(w, utils.BadRequest("account_id is required"))
		return
	}
	if syncRequest.StartDate.IsZero() {
		h.HandleErrors(w, utils.BadRequest("start_date is required"))
		return
	}
	if syncRequest.EndDate.IsZero() {
		h.HandleErrors(w, utils.BadRequest("end_date is required"))
		return
	}
	if syncRequest.EndDate.Before(syncRequest.StartDate) {
		h.HandleErrors(w, utils.BadRequest("end_date must be after start_date"))
		return
	}

	// Get token from request header
	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusBadRequest, "empty token detected"))
		return
	}

	// Call controller to sync account
	accountState, err := h.AccountsController.SyncAccount(context.Background(), token, syncRequest.AccountID, syncRequest.StartDate, syncRequest.EndDate)
	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, accountState, http.StatusOK)
}
