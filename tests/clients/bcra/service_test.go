package bcra_test

import (
	"log"
	"server/src/clients/bcra"
	"server/src/config"
	"testing"
)

func TestBCRAServiceClient(t *testing.T) {
	// Load configuration
	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		log.Println(err, "Error while loading config")
		return
	}

	// Initialize the BCRA service client
	bcraService := bcra.NewClient(cfg)

	t.Run("GetDivisas works correctly", func(t *testing.T) {
		// Call the GetDivisas method
		result, err := bcraService.GetDivisas()

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result.Status != 200 {
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
		result, err := bcraService.GetCotizaciones("2023-09-15")

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result.Status != 200 {
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
		result, err := bcraService.GetCotizacionesPorMoneda("USD", "2023-09-15", "2023-09-16")

		// Check if there was an error
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Check if result is not nil
		if result == nil {
			t.Error("expected result, got nil")
		}

		// Check if the status code is 200
		if result.Status != 200 {
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
}
