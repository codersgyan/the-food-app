package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/codersgyan/the-food-app/internal/service"
	"github.com/codersgyan/the-food-app/internal/store"
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

type orderItemJSON struct {
	DishID   int64  `json:"dish_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

type orderJSON struct {
	ID              int64           `json:"id"`
	RestaurantID    int64           `json:"restaurant_id"`
	Status          string          `json:"status"`
	Total           int             `json:"total"`
	PlacedAt        time.Time       `json:"placed_at"`
	DeliveryAddress string          `json:"delivery_address"`
	Items           []orderItemJSON `json:"items"`
}

func toOrderJSON(o store.Order) orderJSON {
	res := orderJSON{
		ID:              o.ID,
		RestaurantID:    o.RestaurantID,
		Status:          o.Status,
		Total:           o.Total,
		PlacedAt:        o.PlacedAt,
		DeliveryAddress: o.DeliveryAddress,
		Items:           []orderItemJSON{},
	}
	for _, it := range o.Items {
		res.Items = append(res.Items, orderItemJSON{it.DishID, it.Name, it.Quantity, it.Price})
	}
	return res
}

// GET /v1/orders/{id} returns one of the customer's orders, with its status.
func (h *handler) order(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	o, err := h.svc.Orders.Get(r.Context(), claims(r).UserID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrderJSON(o))
}

// GET /v1/restaurants/{id}/orders?status=paid returns the restaurant's orders
// in that status, oldest first. The tablet asks for the paid ones: the
// orders waiting for the kitchen.
func (h *handler) restaurantOrders(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	list, err := h.svc.Orders.ForRestaurant(r.Context(), claims(r).RestaurantID, id, r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, err)
		return
	}

	res := struct {
		Orders []orderJSON `json:"orders"`
	}{Orders: []orderJSON{}}
	for _, o := range list {
		res.Orders = append(res.Orders, toOrderJSON(o))
	}
	writeJSON(w, http.StatusOK, res)
}

// POST /v1/orders/{id}/accept: the kitchen will cook this paid order.
func (h *handler) acceptOrder(w http.ResponseWriter, r *http.Request) {
	h.advanceOrder(w, r, h.svc.Orders.Accept, "accepted")
}

// POST /v1/orders/{id}/ready: the order is cooked and packed.
func (h *handler) readyOrder(w http.ResponseWriter, r *http.Request) {
	h.advanceOrder(w, r, h.svc.Orders.Ready, "ready")
}

func (h *handler) advanceOrder(w http.ResponseWriter, r *http.Request,
	advance func(ctx context.Context, staffRestaurantID, orderID int64) error, status string) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := advance(r.Context(), claims(r).RestaurantID, id); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}{id, status})
}
