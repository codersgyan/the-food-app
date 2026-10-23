package service

import (
	"github.com/codersgyan/the-food-app/internal/auth"
	"github.com/codersgyan/the-food-app/internal/store"
)

// Deps is everything the services need from outside.
type Deps struct {
	Store  *store.Store
	OTP    OTPSender
	Tokens *auth.Tokens
	// NearRadiusKm is how far a restaurant can be and still deliver to a customer.
	NearRadiusKm float64
}

// Services holds one service for each part of the app.
type Services struct {
	Auth        *AuthService
	Users       *UserService
	Restaurants *RestaurantService
}

func New(d Deps) *Services {
	return &Services{
		Auth:        &AuthService{store: d.Store, otp: d.OTP, tokens: d.Tokens},
		Users:       &UserService{store: d.Store},
		Restaurants: &RestaurantService{store: d.Store, nearRadiusKm: d.NearRadiusKm},
	}
}
