package handler

import (
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"

	"github.com/codersgyan/the-food-app/internal/service"
)

// maxBody is the largest request body we read. Our biggest request, an order,
// is a few hundred bytes.
const maxBody = 64 << 10

type errorBody struct {
	Error string `json:"error"`
}

// readJSON decodes the request body into v. A body that isn't valid JSON for v
// becomes a 400.
func readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.UnmarshalRead(r.Body, v); err != nil {
		return &service.Error{Kind: service.ErrInvalid, Msg: "request body is not valid JSON for this request"}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v); err != nil {
		slog.Error("write response", "err", err)
	}
}

// writeError sends a client's mistake back with its status code. Anything
// else is our bug: we log it and send a plain 500, without the details.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var e *service.Error
	if errors.As(err, &e) {
		writeJSON(w, statusFor(e.Kind), errorBody{e.Msg})
		return
	}
	slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeJSON(w, http.StatusInternalServerError, errorBody{"something went wrong"})
}

func statusFor(kind error) int {
	switch kind {
	case service.ErrInvalid:
		return http.StatusBadRequest
	case service.ErrUnauthorized:
		return http.StatusUnauthorized
	case service.ErrForbidden:
		return http.StatusForbidden
	case service.ErrNotFound:
		return http.StatusNotFound
	case service.ErrConflict:
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
