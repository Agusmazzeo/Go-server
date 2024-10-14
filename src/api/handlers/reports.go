package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	"gorm.io/gorm"
)

// HandleGenerateXLSX is the HTTP handler to generate an Excel file
func (h *Handler) GetReportFile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	location, _ := time.LoadLocation("America/Argentina/Buenos_Aires")

	token := jwtauth.TokenFromHeader(r)
	if token == "" {
		h.HandleErrors(w, fmt.Errorf("empty token detected"), http.StatusUnauthorized)
	}

	id := chi.URLParam(r, "id")
	var err error

	startDateStr := r.URL.Query().Get("startDate")
	var startDate time.Time

	endDateStr := r.URL.Query().Get("endDate")
	var endDate time.Time

	format := r.URL.Query().Get("format")

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
	accountState, err := h.AccountsController.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval.ToDuration())
	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if format == "XLSX" {

		xlsxFile, err := h.ReportsController.GetXLSXReport(ctx, accountState, startDate, endDate, interval.ToDuration())
		if err != nil {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}

		// Write the XLSX file to the HTTP response
		err = xlsxFile.Write(w)
		if err != nil {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}
		// Set response headers to download the file
		w.Header().Set("Content-Disposition", "attachment; filename=holdings.xlsx")
	} else {
		df, err := h.ReportsController.GetDataFrameReport(ctx, accountState, startDate, endDate, interval.ToDuration())
		if err != nil {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}
		err = df.WriteCSV(w)
		if err != nil {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}
		// Set response headers to download the file
		w.Header().Set("Content-Disposition", "attachment; filename=holdings.csv")
	}

}

func (h *Handler) GetAllReportSchedules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, err := h.ReportsController.GetAllReportSchedules(ctx)

	if err != nil {
		if err == context.DeadlineExceeded {
			h.HandleErrors(w, err, http.StatusGatewayTimeout)
		} else {
			h.HandleErrors(w, err, http.StatusInternalServerError)
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
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}

	_, err = h.ReportsController.GetReportScheduleByID(ctx, uint(id))

	if err != nil {
		if err == context.DeadlineExceeded {
			h.HandleErrors(w, err, http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			h.HandleErrors(w, err, http.StatusInternalServerError)
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

	created, err := h.ReportsController.CreateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		h.HandleErrors(w, err, http.StatusInternalServerError)
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
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}

	var reportSchedule schemas.UpdateReportScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&reportSchedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reportSchedule.ID = uint(id)

	updated, err := h.ReportsController.UpdateReportSchedule(ctx, &reportSchedule)
	if err != nil {
		if err == context.DeadlineExceeded {
			h.HandleErrors(w, err, http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			h.HandleErrors(w, err, http.StatusInternalServerError)
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
		h.HandleErrors(w, err, http.StatusInternalServerError)
		return
	}

	err = h.ReportsController.DeleteReportSchedule(ctx, uint(id))
	if err != nil {
		if err == context.DeadlineExceeded {
			h.HandleErrors(w, err, http.StatusGatewayTimeout)
		} else if err == gorm.ErrRecordNotFound {
			http.Error(w, "Report schedule not found", http.StatusNotFound)
		} else {
			h.HandleErrors(w, err, http.StatusInternalServerError)
		}
		return
	}

	h.respond(w, r, nil, http.StatusNoContent)
}
