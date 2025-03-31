package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
)

// HandleGenerateXLSX is the HTTP handler to generate an Excel file
func (h *Handler) GetReportByIDs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	ctx = utils.WithLogger(ctx, h.Logger)

	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnauthorized, "auth token not detected"))
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
		h.Logger.Warning(err)
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
		return
	}

	dailyInterval, _ := utils.ParseTimeInterval("0w:1d")

	startDate, err = time.Parse(utils.ShortDashDateLayout, startDateStr)
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
		return
	}
	endDate, err = time.Parse(utils.ShortDashDateLayout, endDateStr)
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
		return
	}
	//Set +26 hours since we use ARG timezone (UTC-3)
	startDate = (startDate.Add(26 * time.Hour)).In(location)
	endDate = (endDate.Add(26 * time.Hour)).In(location)
	accountsStates, err := h.AccountsController.GetMultiAccountStateByCategoryDateRange(ctx, token, ids, startDate, endDate, dailyInterval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	referenceVariables, err := h.Controller.GetReferenceVariablesWithValuationDateRange(ctx, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	accountsReports, err := h.ReportsController.GetReport(ctx, accountsStates, referenceVariables, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	h.respond(w, r, accountsReports, 200)
}

// HandleGenerateXLSX is the HTTP handler to generate an Excel file
func (h *Handler) GetReportFile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnauthorized, "auth token not detected"))
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
	startDateStr := r.URL.Query().Get("startDate")
	var startDate time.Time

	endDateStr := r.URL.Query().Get("endDate")
	var endDate time.Time

	format := r.URL.Query().Get("format")

	intervalStr := r.URL.Query().Get("interval")
	if intervalStr == "" {
		// Set interval per day as default
		intervalStr = "0w:1d"
	}
	interval, err := utils.ParseTimeInterval(intervalStr)
	if err != nil {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
		return
	}

	dailyInterval, _ := utils.ParseTimeInterval("0w:1d")

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
	accountsStates, err := h.AccountsController.GetMultiAccountStateByCategoryDateRange(ctx, token, ids, startDate, endDate, dailyInterval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	referenceVariables, err := h.Controller.GetReferenceVariablesWithValuationDateRange(ctx, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	accountsReports, err := h.ReportsController.GetReport(ctx, accountsStates, referenceVariables, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}

	dataframes, err := h.ReportsController.ParseAccountsReportToDataFrames(ctx, accountsReports, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.Logger.Warning(err)
		h.HandleErrors(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if format == "XLSX" {

		xlsxFile, err := h.ReportsController.ParseAccountsReportToXLSX(ctx, dataframes)
		if err != nil {
			h.HandleErrors(w, err)
			return
		}

		// Write the XLSX file to the HTTP response
		err = xlsxFile.Write(w)
		if err != nil {
			h.HandleErrors(w, err)
			return
		}
		// Set response headers to download the file
		w.Header().Set("Content-Disposition", "attachment; filename=holdings.xlsx")
	} else {
		pdfData, err := h.ReportsController.ParseAccountsReportToPDF(ctx, dataframes)
		if err != nil {
			h.HandleErrors(w, err)
			return
		}
		// Set headers for PDF download
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=report.pdf")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfData)))
		// Write the PDF data to the response
		_, err = w.Write(pdfData)
		if err != nil {
			h.HandleErrors(w, err)
			return
		}
	}

}

func (h *Handler) GetAllReportSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.ReportsController.GetAllReportSchedules(ctx)

	if err != nil {
		h.HandleErrors(w, err)
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
		h.HandleErrors(w, utils.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
		return
	}

	_, err = h.ReportsController.GetReportScheduleByID(ctx, uint(id))

	if err != nil {
		h.HandleErrors(w, err)
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

	created, err := h.ReportsController.CreateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		h.HandleErrors(w, err)
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
		h.HandleErrors(w, err)
		return
	}

	var reportSchedule schemas.UpdateReportScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&reportSchedule); err != nil {
		h.HandleErrors(w, utils.NewHTTPError(http.StatusBadRequest, err.Error()))
		return
	}

	reportSchedule.ID = uint(id)

	updated, err := h.ReportsController.UpdateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		h.HandleErrors(w, err)
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
		h.HandleErrors(w, err)
		return
	}

	err = h.ReportsController.DeleteReportSchedule(ctx, uint(id))
	if err != nil {
		h.HandleErrors(w, err)
		return
	}

	h.respond(w, r, nil, http.StatusNoContent)
}
