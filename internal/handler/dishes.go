package handler

import (
	"net/http"

	"github.com/codersgyan/the-food-app/internal/service"
)

// PATCH /v1/dishes/{id} changes only the fields sent, like {"sold_out": true}.
func (h *handler) updateDish(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	// Pointers tell "not sent" (nil) apart from "sent as false or 0".
	var req struct {
		Price   *int  `json:"price"`
		SoldOut *bool `json:"sold_out"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	d, err := h.svc.Dishes.Update(r.Context(), claims(r).RestaurantID, id, service.DishChanges{Price: req.Price, SoldOut: req.SoldOut})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dishJSON{ID: d.ID, Name: d.Name, Price: d.Price, SoldOut: d.SoldOut})
}
