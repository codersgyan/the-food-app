package handler

import (
	"net/http"
	"strconv"

	"github.com/codersgyan/the-food-app/internal/service"
)

// pathID reads a numeric ID from the path, like the 42 in /v1/restaurants/42/menu.
func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, &service.Error{Kind: service.ErrNotFound, Msg: name + " must be a positive number"}
	}
	return id, nil
}
