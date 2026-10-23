package service

import (
	"context"

	"github.com/codersgyan/the-food-app/internal/store"
)

// RestaurantService helps customers find restaurants and see their menus.
type RestaurantService struct {
	store *store.Store
	// nearRadiusKm is how far a restaurant can be and still deliver to a customer.
	nearRadiusKm float64
}

// Near returns the restaurants that deliver to a customer at lat, lng, nearest first.
func (s *RestaurantService) Near(ctx context.Context, lat, lng float64) ([]store.Restaurant, error) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, fail(ErrInvalid, "near must be a real spot on the map")
	}
	return s.store.RestaurantsNear(ctx, lat, lng, s.nearRadiusKm)
}
