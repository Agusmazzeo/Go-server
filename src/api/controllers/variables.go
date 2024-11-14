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
	response, err := c.BCRAClient.GetVariablePorFecha(ctx, id, date.Format("2006-01-02"), date.AddDate(0, 0, 1).Format("2006-01-02"))
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

func (c *Controller) GetReferenceVariablesWithValuationDateRange(ctx context.Context, startDate, endDate time.Time) ([]*schemas.VariableWithValuationResponse, error) {
	var wg sync.WaitGroup
	var variableValuations = make([]*schemas.VariableWithValuationResponse, 0, 2)
	var variableValuationsChan = make(chan *schemas.VariableWithValuationResponse)
	var errChan = make(chan error)
	wg.Add(2)
	go func() {
		defer wg.Done()
		variableValuations, err := c.GetA3500DateRange(ctx, startDate, endDate)
		if err != nil {
			errChan <- err
			return
		}
		variableValuationsChan <- variableValuations
	}()
	go func() {
		defer wg.Done()
		variableValuations, err := c.GetMonthlyInflationDateRange(ctx, startDate, endDate)
		if err != nil {
			errChan <- err
			return
		}
		variableValuationsChan <- variableValuations
	}()
	go func() {
		wg.Wait()
		variableValuationsChan <- nil
	}()
	for {
		select {
		case err := <-errChan:
			return nil, err
		case variableValuation := <-variableValuationsChan:
			if variableValuation == nil {
				return variableValuations, nil
			}
			variableValuations = append(variableValuations, variableValuation)
		}
	}
}

func (c *Controller) GetVariableWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error) {
	variablesMap, err := c.getVariablesMap(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.BCRAClient.GetVariablePorFecha(ctx, id, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
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
	return c.GetVariableWithValuationDateRangeByID(ctx, utils.A3500ID, startDate, endDate)
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
