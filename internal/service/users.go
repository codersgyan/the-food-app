package service

import (
	"context"

	"github.com/codersgyan/the-food-app/internal/store"
)

// UserService is about the logged-in customer: who they are and where they live.
type UserService struct {
	store *store.Store
}

// Me returns the user and their saved addresses.
func (s *UserService) Me(ctx context.Context, userID int64) (store.User, []store.Address, error) {
	u, err := s.store.User(ctx, userID)
	if err != nil {
		return store.User{}, nil, err
	}
	addresses, err := s.store.AddressesOf(ctx, userID)
	if err != nil {
		return store.User{}, nil, err
	}
	return u, addresses, nil
}
