package service

import (
	"context"
	"errors"

	"github.com/codersgyan/the-food-app/internal/store"
)

// DishService lets restaurant staff change their menu.
type DishService struct {
	store *store.Store
}

// DishChanges holds the fields to change. A nil field stays as it is.
type DishChanges struct {
	Price   *int // in paise
	SoldOut *bool
}

// Update changes one of the restaurant's dishes. Orders already placed keep
// the price they were placed at, because order_items has its own copy.
func (s *DishService) Update(ctx context.Context, staffRestaurantID, dishID int64, c DishChanges) (store.Dish, error) {
	if c.Price == nil && c.SoldOut == nil {
		return store.Dish{}, fail(ErrInvalid, "send price, sold_out, or both")
	}
	if c.Price != nil && *c.Price <= 0 {
		return store.Dish{}, fail(ErrInvalid, "price must be more than 0 paise")
	}
	d, err := s.store.UpdateDish(ctx, staffRestaurantID, dishID, c.Price, c.SoldOut)
	if errors.Is(err, store.ErrNotFound) {
		return store.Dish{}, fail(ErrNotFound, "dish %d not found on your menu", dishID)
	}
	return d, err
}
