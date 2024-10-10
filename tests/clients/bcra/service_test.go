package bcra_test

import (
	"context"
	"testing"
)

func TestBCRAServiceClient(t *testing.T) {
	// t.Skip("BCRA API calls fail in the CI agent")
	// Load configuration
	// cfg, err := config.LoadConfig("../../../settings")
	// if err != nil {
	// 	log.Println(err, "Error while loading config")
	// 	return
	// }

	// // Initialize the BCRA service client
	// bcraService, err := bcra.NewClient(cfg)
	// if err != nil {
	// 	t.Errorf("expected no error, got %v", err)
	// }

	bcraService, _ := NewMockClient("../../test_files/clients/bcra")

	t.Run("GetDivisas works correctly", func(t *testing.T) {
		// Call the GetDivisas method
		result, err := bcraService.GetDivisas(context.Background())

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result != nil && result.Status != 200 {
			t.Errorf("expected status 200, got %d", result.Status)
		}

		// Check if the results contain divisas
		if len(result.Results) == 0 {
			t.Error("expected some divisas in results, got none")
		}

		// Optionally, check specific divisas
		if result.Results[0].Codigo != "ARS" {
			t.Errorf("expected first divisa to be ARS, got %s", result.Results[0].Codigo)
		}
		if result.Results[0].Denominacion != "PESO" {
			t.Errorf("expected first divisa to be PESO, got %s", result.Results[0].Denominacion)
		}
	})

	t.Run("GetCotizaciones works correctly", func(t *testing.T) {
		// Call the GetCotizaciones method with a valid date
		result, err := bcraService.GetCotizaciones(context.Background(), "2023-09-15")

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result != nil && result.Status != 200 {
			t.Errorf("expected status 200, got %d", result.Status)
		}

		// Check if the results contain cotizaciones
		if len(result.Results.Detalle) == 0 {
			t.Error("expected some cotizaciones in results, got none")
		}

		// Optionally, check specific cotizacion details
		if result.Results.Detalle[0].CodigoMoneda != "ARS" {
			t.Errorf("expected first cotizacion to be ARS, got %s", result.Results.Detalle[0].CodigoMoneda)
		}
	})

	t.Run("GetCotizacionesPorMoneda works correctly", func(t *testing.T) {
		// Call the GetCotizacionesPorMoneda method with valid params
		result, err := bcraService.GetCotizacionesPorMoneda(context.Background(), "USD", "2023-09-15", "2023-09-16")

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result != nil && result.Status != 200 {
			t.Errorf("expected status 200, got %d", result.Status)
		}

		// Check if the results contain cotizaciones for the given currency
		if len(result.Results) == 0 {
			t.Error("expected some cotizaciones for USD, got none")
		}

		// Optionally, check specific cotizacion details
		if result.Results[0].Detalle[0].CodigoMoneda != "USD" {
			t.Errorf("expected first cotizacion to be USD, got %s", result.Results[0].Detalle[0].CodigoMoneda)
		}
	})

	t.Run("GetVariables works correctly", func(t *testing.T) {
		// Call the GetCotizacionesPorMoneda method with valid params
		result, err := bcraService.GetVariables(context.Background())

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result != nil && result.Status != 200 {
			t.Errorf("expected status 200, got %d", result.Status)
		}

		// Check if the results contain cotizaciones for the given currency
		if len(result.Results) == 0 {
			t.Error("expected some cotizaciones for USD, got none")
		}

		// Optionally, check specific cotizacion details
		if result.Results[2].Descripcion != "Tipo de Cambio Mayorista ($ por USD) Comunicación A 3500\u00A0- Referencia" {
			t.Errorf("expected first cotizacion to be USD, got %s", result.Results[2].Descripcion)
		}
	})

	t.Run("GetVariablePorFecha works correctly", func(t *testing.T) {
		// Call the GetCotizacionesPorMoneda method with valid params
		result, err := bcraService.GetVariablePorFecha(context.Background(), "13", "2024-01-01", "2024-01-02")

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result != nil && result.Status != 200 {
			t.Errorf("expected status 200, got %d", result.Status)
		}

		// Check if the results contain variables for the given id
		if len(result.Results) == 0 {
			t.Error("expected some variables, got none")
		}

		// Optionally, check specific cotizacion details
		if result.Results[0].IDVariable != 13 {
			t.Errorf("expected id to be 13, got %s", result.Results[2].Descripcion)
		}
	})
}
