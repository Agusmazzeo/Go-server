package handlers_test

import (
	"io"
	"net/http"
	"os"
	"testing"
)

func TestGetReportXLSX(t *testing.T) {
	// Create a request for the XLSX report
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/reports/11170?startDate=2024-08-01&endDate=2024-08-03&format=XLSX", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	// Send the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// Check that the response status is OK
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK; got %v", res.Status)
	}

	// Check if the content type is "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" (XLSX)
	contentType := res.Header.Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("expected content type application/vnd.openxmlformats-officedocument.spreadsheetml.sheet; got %v", contentType)
	}

	// Create a file to save the XLSX data
	outFile, err := os.Create("../../test_files/handlers/reports/report.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()

	// Write the response body to the file
	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the file was downloaded and has content
	fileInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if fileInfo.Size() == 0 {
		t.Fatalf("downloaded file is empty")
	}

	t.Logf("XLSX report downloaded successfully, file size: %d bytes", fileInfo.Size())
}

func TestGetReportPDF(t *testing.T) {
	// Create a request for the XLSX report
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/reports/11170?startDate=2024-08-01&endDate=2024-08-03&format=PDF", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	// Send the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// Check that the response status is OK
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK; got %v", res.Status)
	}

	// Check if the content type is "application/pdf" (PDF)
	contentType := res.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		t.Fatalf("expected content type application/pdf; got %v", contentType)
	}

	// Create a file to save the XLSX data
	outFile, err := os.Create("../../test_files/handlers/reports/report.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()

	// Write the response body to the file
	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the file was downloaded and has content
	fileInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if fileInfo.Size() == 0 {
		t.Fatalf("downloaded file is empty")
	}

	t.Logf("PDF report downloaded successfully, file size: %d bytes", fileInfo.Size())
}
