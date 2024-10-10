package controllers

import (
	"context"
	"server/src/schemas"
	"strconv"
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

func (c *Controller) GetVariableWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error) {
	variablesMap, err := c.getVariablesMap(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.BCRAClient.GetVariablePorFecha(ctx, id, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
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
