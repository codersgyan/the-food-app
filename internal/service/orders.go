package service

import (
	"context"
	"errors"

	"github.com/codersgyan/the-food-app/internal/store"
)

// OrderService places orders and moves them through their statuses.
type OrderService struct {
	store *store.Store
}

// NewOrder is what the app sends to place an order. It has no prices: the
// server reads them from the menu.
type NewOrder struct {
	RestaurantID int64
	AddressID    int64
	Items        []NewOrderItem
}

type NewOrderItem struct {
	DishID   int64
	Quantity int
}

// Place saves a new order for the user, in the pending status.
func (s *OrderService) Place(ctx context.Context, userID int64, in NewOrder) (store.Order, error) {
	if len(in.Items) == 0 || len(in.Items) > 50 {
		return store.Order{}, fail(ErrInvalid, "an order has 1 to 50 dishes")
	}
	ids := make([]int64, 0, len(in.Items))
	seen := make(map[int64]bool)
	for _, it := range in.Items {
		if it.Quantity < 1 || it.Quantity > 20 {
			return store.Order{}, fail(ErrInvalid, "quantity must be 1 to 20")
		}
		if seen[it.DishID] {
			return store.Order{}, fail(ErrInvalid, "dish %d is in the order twice, send it once with its quantity", it.DishID)
		}
		seen[it.DishID] = true
		ids = append(ids, it.DishID)
	}

	var order store.Order
	err := s.store.InTx(ctx, func(tx *store.Store) error {
		addr, err := tx.Address(ctx, in.AddressID, userID)
		if errors.Is(err, store.ErrNotFound) {
			return fail(ErrInvalid, "address %d is not one of your saved addresses", in.AddressID)
		}
		if err != nil {
			return err
		}

		dishes, err := tx.RestaurantDishes(ctx, in.RestaurantID, ids)
		if err != nil {
			return err
		}
		menu := make(map[int64]store.Dish, len(dishes))
		for _, d := range dishes {
			menu[d.ID] = d
		}

		order = store.Order{
			UserID:          userID,
			RestaurantID:    in.RestaurantID,
			DeliveryAddress: addr.Address,
		}
		for _, it := range in.Items {
			d, ok := menu[it.DishID]
			if !ok {
				return fail(ErrInvalid, "dish %d is not on restaurant %d's menu", it.DishID, in.RestaurantID)
			}
			if d.SoldOut {
				return fail(ErrConflict, "%s is sold out", d.Name)
			}
			// The price comes from our menu, never from the app. A price sent
			// from the phone could have been changed on the way.
			order.Items = append(order.Items, store.OrderItem{
				DishID: d.ID, Name: d.Name, Quantity: it.Quantity, Price: d.Price,
			})
			order.Total += d.Price * it.Quantity
		}

		order, err = tx.InsertOrder(ctx, order)
		return err
	})
	return order, err
}

// Get returns one of the user's orders. Someone else's order looks the same
// as one that doesn't exist, so nobody can find out which order IDs are taken.
func (s *OrderService) Get(ctx context.Context, userID, orderID int64) (store.Order, error) {
	o, err := s.store.Order(ctx, orderID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && o.UserID != userID) {
		return store.Order{}, fail(ErrNotFound, "order %d not found", orderID)
	}
	return o, err
}

var statuses = map[string]bool{"pending": true, "paying": true, "paid": true, "accepted": true, "ready": true}

// ForRestaurant returns a restaurant's orders in one status. Staff can see
// only their own restaurant's orders.
func (s *OrderService) ForRestaurant(ctx context.Context, staffRestaurantID, restaurantID int64, status string) ([]store.Order, error) {
	if staffRestaurantID != restaurantID {
		return nil, fail(ErrForbidden, "you can only see your own restaurant's orders")
	}
	if !statuses[status] {
		return nil, fail(ErrInvalid, "status must be pending, paying, paid, accepted or ready")
	}
	return s.store.RestaurantOrders(ctx, restaurantID, status)
}
