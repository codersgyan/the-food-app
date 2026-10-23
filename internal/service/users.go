package service

import (
	"context"
	"strings"

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

// AddAddress saves a delivery address for the user and returns its ID.
func (s *UserService) AddAddress(ctx context.Context, userID int64, address string) (int64, error) {
	address = strings.TrimSpace(address)
	if address == "" || len(address) > 300 {
		return 0, fail(ErrInvalid, "address must be 1 to 300 characters")
	}
	return s.store.AddAddress(ctx, userID, address)
}
