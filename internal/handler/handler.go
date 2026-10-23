// Package handler turns HTTP requests into service calls, and service results
// into HTTP responses. It holds no SQL and no business rules.
package handler

import (
	"net/http"

	"github.com/codersgyan/the-food-app/internal/auth"
	"github.com/codersgyan/the-food-app/internal/service"
)

type handler struct {
	svc    *service.Services
	tokens *auth.Tokens
}

// New returns the router with every route of the API.
func New(svc *service.Services, tokens *auth.Tokens) http.Handler {
	h := &handler{svc: svc, tokens: tokens}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/login", h.login)
	mux.HandleFunc("POST /v1/login/verify", h.verifyLogin)
	mux.HandleFunc("GET /v1/me", h.customer(h.me))
	mux.HandleFunc("POST /v1/addresses", h.customer(h.addAddress))
	mux.HandleFunc("GET /v1/restaurants", h.restaurantsNear)
	mux.HandleFunc("GET /v1/restaurants/{id}/menu", h.menu)
	mux.HandleFunc("POST /v1/orders", h.customer(h.placeOrder))
	mux.HandleFunc("GET /v1/orders/{id}", h.customer(h.order))

	return logRequests(recoverPanics(mux))
}
