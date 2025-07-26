package controllers

import (
	"context"
	"fmt"
	"net/http"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"sync"
	"time"
)

func (c *Controller) GetAllVariables(ctx context.Context) ([]schemas.Variable, error) {
	response, err := c.BCRAClient.GetVariables(ctx)
	if err != nil {
		return nil, err
	}

	variables := make([]schemas.Variable, 0, len(response.Results))
	for _, variable := range response.Results {
		variables = append(variables, schemas.Variable{ID: strconv.Itoa(variable.IDVariable), Description: variable.Descripcion})
	}
	return variables, nil
}

func (c *Controller) GetVariableWithValuationByID(ctx context.Context, id string, date time.Time) (*schemas.VariableWithValuationResponse, error) {
	variablesMap, err := c.getVariablesMap(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.BCRAClient.GetVariablesPorFecha(ctx, id, date.Format("2006-01-02"), date.AddDate(0, 0, 1).Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	if response.Status != http.StatusOK {
		return nil, utils.NewHTTPError(response.Status, fmt.Sprintf("Error requesting BCRA variables: %s", response.ErrorMessages[0]))
	}
	if len(response.Results) == 0 {
		return nil, nil
	}
	varID := strconv.Itoa(response.Results[0].IDVariable)
	variableResponse := &schemas.VariableWithValuationResponse{
		ID:          varID,
		Description: variablesMap[varID],
		Valuations:  make([]schemas.VariableValuation, 0, len(response.Results)),
	}
	for _, variable := range response.Results {
		variableResponse.Valuations = append(variableResponse.Valuations, schemas.VariableValuation{
			Date:  variable.Fecha,
			Value: variable.Valor,
		})

	}
	return variableResponse, nil
}

// GetReferenceVariablesWithValuationDateRange works for getting reference variables in a date range
func (c *Controller) GetReferenceVariablesWithValuationDateRange(ctx context.Context, startDate, endDate time.Time, interval time.Duration) (map[string]*schemas.VariableWithValuationResponse, error) {
	var wg sync.WaitGroup
	var variableValuationsMap = map[string]*schemas.VariableWithValuationResponse{}
	var errChan = make(chan error)
	wg.Add(2)
	go func() {
		defer wg.Done()
		variableValuations, err := c.GetA3500DateRange(ctx, startDate, endDate)
		if err != nil {
			errChan <- err
			return
		}
		variableValuationsMap["USD A3500"] = variableValuations
		variableValuationsMap["USD A3500 Variacion"] = ComputeValuationVariations(variableValuations, interval)
	}()
	go func() {
		defer wg.Done()
		variableValuations, err := c.GetMonthlyInflationDateRange(ctx, startDate, endDate)
		if err != nil {
			errChan <- err
			return
		}
		variableValuationsMap["Inflacion Mensual"] = variableValuations
	}()
	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err := <-errChan; err != nil {
		return nil, err
	}
	return variableValuationsMap, nil
}

func (c *Controller) GetVariableWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error) {
	variablesMap, err := c.getVariablesMap(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.BCRAClient.GetVariablesPorFecha(ctx, id, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	if response.Status != http.StatusOK {
		return nil, utils.NewHTTPError(response.Status, fmt.Sprintf("Error requesting BCRA variables: %s", response.ErrorMessages[0]))
	}
	if len(response.Results) == 0 {
		return nil, nil
	}
	varID := strconv.Itoa(response.Results[0].IDVariable)
	variableResponse := &schemas.VariableWithValuationResponse{
		ID:          varID,
		Description: variablesMap[varID],
		Valuations:  make([]schemas.VariableValuation, 0, len(response.Results)),
	}
	for _, variable := range response.Results {
		variableResponse.Valuations = append(variableResponse.Valuations, schemas.VariableValuation{
			Date:  variable.Fecha,
			Value: variable.Valor,
		})

	}
	return variableResponse, nil
}

func (c *Controller) GetMonthlyInflationDateRange(ctx context.Context, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error) {
	startDate = startDate.Add(-30 * 24 * time.Hour)
	variableWithValuations, err := c.GetVariableWithValuationDateRangeByID(ctx, utils.MonthlyInflationID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	err = c.CompleteValuations(variableWithValuations, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return variableWithValuations, nil
}

// Function to complete missing days and fill values
func (c *Controller) CompleteValuations(v *schemas.VariableWithValuationResponse, startDate, endDate time.Time) error {
	// Sort valuations by date
	sort.Slice(v.Valuations, func(i, j int) bool {
		dateI, _ := time.Parse("2006-01-02", v.Valuations[i].Date)
		dateJ, _ := time.Parse("2006-01-02", v.Valuations[j].Date)
		return dateI.Before(dateJ)
	})

	// Create a map to quickly access valuations by date
	valuationMap := make(map[string]float64)
	for _, valuation := range v.Valuations {
		valuationMap[valuation.Date] = valuation.Value
	}

	// Prepare the result slice and fill in missing days
	var filledValuations []schemas.VariableValuation
	var lastKnownValue float64

	currentDate := startDate
	for !currentDate.After(endDate) {
		dateStr := currentDate.Format("2006-01-02")

		// Check if the current date exists in the map
		value, exists := valuationMap[dateStr]
		if exists {
			// Update the last known value if the date has a recorded valuation
			lastKnownValue = value
		} else {
			// Use the last known value if date has no recorded valuation
			value = lastKnownValue
		}
		if value == 0 {
			// Move to the next day
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		// Append the valuation for the current date
		filledValuations = append(filledValuations, schemas.VariableValuation{
			Date:  dateStr,
			Value: value,
		})

		// Move to the next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}
	sort.Slice(filledValuations, func(i, j int) bool {
		dateI, _ := time.Parse("2006-01-02", filledValuations[i].Date)
		dateJ, _ := time.Parse("2006-01-02", filledValuations[j].Date)
		return dateI.Before(dateJ)
	})
	// Update the original valuations slice with the filled data
	v.Valuations = filledValuations
	return nil
}

func (c *Controller) GetA3500DateRange(ctx context.Context, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error) {
	a3500Valuations, err := c.GetVariableWithValuationDateRangeByID(ctx, utils.A3500ID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	err = c.CompleteValuations(a3500Valuations, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return a3500Valuations, nil
}

func (c *Controller) getVariablesMap(ctx context.Context) (map[string]string, error) {
	variables, err := c.GetAllVariables(ctx)
	if err != nil {
		return nil, err
	}
	variablesMap := map[string]string{}
	for _, variable := range variables {
		variablesMap[variable.ID] = variable.Description
	}
	return variablesMap, nil
}

// ComputeValuationVariations works for calculating variables reference variation over time
func ComputeValuationVariations(input *schemas.VariableWithValuationResponse, interval time.Duration) *schemas.VariableWithValuationResponse {
	if len(input.Valuations) == 0 {
		return &schemas.VariableWithValuationResponse{
			ID:          input.ID,
			Description: input.Description + " (variation)",
			Valuations:  []schemas.VariableValuation{},
		}
	}

	// Parse and map valuations by date
	valuationMap := make(map[time.Time]float64)
	var dateList []time.Time

	for _, val := range input.Valuations {
		t, err := time.Parse("2006-01-02", val.Date)
		if err != nil {
			continue // skip invalid date
		}
		valuationMap[t] = val.Value
		dateList = append(dateList, t)
	}

	// Sort dates
	sort.Slice(dateList, func(i, j int) bool {
		return dateList[i].Before(dateList[j])
	})

	// Compute variations over interval
	var variations []schemas.VariableValuation
	for _, date := range dateList {
		prevDate := date.Add(-interval)
		prevVal, ok := valuationMap[prevDate]
		currVal := valuationMap[date]

		if ok {
			delta := (currVal - prevVal) / prevVal
			variations = append(variations, schemas.VariableValuation{
				Date:  date.Format("2006-01-02"),
				Value: delta,
			})
		}
	}

	return &schemas.VariableWithValuationResponse{
		ID:          input.ID,
		Description: input.Description + " (variation)",
		Valuations:  variations,
	}
}
