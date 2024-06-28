package handlers

import (
	"fmt"
	"net/http"
)

func Healthcheck(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprintf(w, "Im alive!")
	} else {
		fmt.Fprintf(w, "Method not available: %s", r.Method)
	}
}
