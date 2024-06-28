package handlers

import (
	"fmt"
	"net/http"
)

func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Fprintf(w, "Im alive!")
	}
}
