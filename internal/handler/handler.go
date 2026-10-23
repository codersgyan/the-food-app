// Package handler turns HTTP requests into service calls, and service results
// into HTTP responses. It holds no SQL and no business rules.
package handler

import "net/http"

// New returns the router with every route of the API.
func New() http.Handler {
	mux := http.NewServeMux()
	return logRequests(recoverPanics(mux))
}
