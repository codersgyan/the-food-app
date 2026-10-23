package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/codersgyan/the-food-app/internal/service"
)

// POST /v1/orders places an order. The body has no prices: the server reads
// them from the menu.
func (h *handler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RestaurantID int64 `json:"restaurant_id"`
		AddressID    int64 `json:"address_id"`
		Items        []struct {
			DishID   int64 `json:"dish_id"`
			Quantity int   `json:"quantity"`
		} `json:"items"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	in := service.NewOrder{RestaurantID: req.RestaurantID, AddressID: req.AddressID}
	for _, it := range req.Items {
		in.Items = append(in.Items, service.NewOrderItem{DishID: it.DishID, Quantity: it.Quantity})
	}

	o, err := h.svc.Orders.Place(r.Context(), claims(r).UserID, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/v1/orders/%d", o.ID))
	writeJSON(w, http.StatusCreated, struct {
		ID       int64     `json:"id"`
		Status   string    `json:"status"`
		Total    int       `json:"total"`
		PlacedAt time.Time `json:"placed_at"`
	}{o.ID, o.Status, o.Total, o.PlacedAt})
}
