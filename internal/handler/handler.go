// Package handler turns HTTP requests into service calls, and service results
// into HTTP responses. It holds no SQL and no business rules.
package handler

import (
	"net/http"

	"github.com/codersgyan/the-food-app/internal/service"
)

type handler struct {
	svc *service.Services
}

// New returns the router with every route of the API.
func New(svc *service.Services) http.Handler {
	h := &handler{svc: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/login", h.login)

	return logRequests(recoverPanics(mux))
}
