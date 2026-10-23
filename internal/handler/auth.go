package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/codersgyan/the-food-app/internal/auth"
)

type claimsKey struct{}

// customer lets a request through only with a valid customer's token.
func (h *handler) customer(next http.HandlerFunc) http.HandlerFunc {
	return h.requireRole(auth.RoleCustomer, next)
}

// restaurant lets a request through only with a valid restaurant staff member's token.
func (h *handler) restaurant(next http.HandlerFunc) http.HandlerFunc {
	return h.requireRole(auth.RoleRestaurant, next)
}

func (h *handler) requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorBody{"log in first"})
			return
		}
		c, err := h.tokens.Parse(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorBody{"token is invalid or expired, log in again"})
			return
		}
		if c.Role != role {
			writeJSON(w, http.StatusForbidden, errorBody{"only " + role + " accounts can do this"})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, c)))
	}
}

// claims returns the logged-in user of a request that passed customer or restaurant.
func claims(r *http.Request) auth.Claims {
	return r.Context().Value(claimsKey{}).(auth.Claims)
}
