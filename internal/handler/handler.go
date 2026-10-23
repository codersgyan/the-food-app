// Package handler turns HTTP requests into service calls, and service results
// into HTTP responses. It holds no SQL and no business rules.
package handler

import (
	"net/http"

	"github.com/codersgyan/the-food-app/internal/auth"
	"github.com/codersgyan/the-food-app/internal/service"
)

type handler struct {
	svc           *service.Services
	tokens        *auth.Tokens
	webhookSecret string
}

// New returns the router with every route of the API. webhookSecret is the
// secret the payment provider signs its webhooks with.
func New(svc *service.Services, tokens *auth.Tokens, webhookSecret string) http.Handler {
	h := &handler{svc: svc, tokens: tokens, webhookSecret: webhookSecret}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/login", h.login)
	mux.HandleFunc("POST /v1/login/verify", h.verifyLogin)
	mux.HandleFunc("GET /v1/me", h.customer(h.me))
	mux.HandleFunc("POST /v1/addresses", h.customer(h.addAddress))
	mux.HandleFunc("GET /v1/restaurants", h.restaurantsNear)
	mux.HandleFunc("GET /v1/restaurants/{id}/menu", h.menu)
	mux.HandleFunc("POST /v1/orders", h.customer(h.placeOrder))
	mux.HandleFunc("GET /v1/orders/{id}", h.customer(h.order))
	mux.HandleFunc("POST /v1/orders/{id}/pay", h.customer(h.pay))
	mux.HandleFunc("POST /payments/webhook", h.paymentWebhook)

	// The restaurant's tablet
	mux.HandleFunc("GET /v1/restaurants/{id}/orders", h.restaurant(h.restaurantOrders))

	return logRequests(recoverPanics(mux))
}
