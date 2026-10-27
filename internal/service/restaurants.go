package service

import (
	"context"
	"errors"

	"github.com/codersgyan/the-food-app/internal/store"
)

// RestaurantService helps customers find restaurants and see their menus.
type RestaurantService struct {
	store *store.Store
	// nearRadiusKm is how far a restaurant can be and still deliver to a customer.
	nearRadiusKm float64
}

// restaurantsPerPage is how many restaurants one page of Near has: about
// what a phone screen shows, so the app asks for the next page as the
// customer scrolls.
const restaurantsPerPage = 20

// Near returns one page of the restaurants that deliver to a customer at
// lat, lng, nearest first. Page 1 is the nearest 20.
func (s *RestaurantService) Near(ctx context.Context, lat, lng float64, page int) ([]store.Restaurant, error) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, fail(ErrInvalid, "near must be a real spot on the map")
	}
	if page < 1 {
		return nil, fail(ErrInvalid, "page must be 1 or more")
	}
	return s.store.RestaurantsNear(ctx, lat, lng, s.nearRadiusKm, restaurantsPerPage, (page-1)*restaurantsPerPage)
}

// Menu returns a restaurant and all its dishes, including the sold-out ones,
// so the app can show them with a "sold out" label.
func (s *RestaurantService) Menu(ctx context.Context, restaurantID int64) (store.Restaurant, []store.Dish, error) {
	r, err := s.store.Restaurant(ctx, restaurantID)
	if errors.Is(err, store.ErrNotFound) {
		return store.Restaurant{}, nil, fail(ErrNotFound, "restaurant %d not found", restaurantID)
	}
	if err != nil {
		return store.Restaurant{}, nil, err
	}
	dishes, err := s.store.DishesOf(ctx, restaurantID)
	if err != nil {
		return store.Restaurant{}, nil, err
	}
	return r, dishes, nil
}
