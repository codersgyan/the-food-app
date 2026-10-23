package service

import (
	"context"
	"regexp"

	"github.com/codersgyan/the-food-app/internal/auth"
	"github.com/codersgyan/the-food-app/internal/store"
)

// OTPSender texts a one-time login code to a phone, and checks the code the
// user types. The OTP service stores the codes, so we don't.
type OTPSender interface {
	Send(ctx context.Context, phone string) error
	Check(ctx context.Context, phone, code string) (bool, error)
}

// AuthService logs users in with their phone number and a code texted to it.
type AuthService struct {
	store  *store.Store
	otp    OTPSender
	tokens *auth.Tokens
}

// A phone number in international format, like +919845012345.
var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// StartLogin texts a login code to phone.
func (s *AuthService) StartLogin(ctx context.Context, phone string) error {
	if !phonePattern.MatchString(phone) {
		return fail(ErrInvalid, "phone must be in international format, like +919845012345")
	}
	return s.otp.Send(ctx, phone)
}

// VerifyLogin checks the code texted to phone and returns a token for the
// user. The first login with a new phone number creates a customer.
func (s *AuthService) VerifyLogin(ctx context.Context, phone, code string) (string, error) {
	if !phonePattern.MatchString(phone) {
		return "", fail(ErrInvalid, "phone must be in international format, like +919845012345")
	}
	ok, err := s.otp.Check(ctx, phone, code)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fail(ErrUnauthorized, "wrong code")
	}

	u, err := s.store.UserByPhoneOrNew(ctx, phone)
	if err != nil {
		return "", err
	}
	c := auth.Claims{UserID: u.ID, Role: u.Role}
	if u.RestaurantID != nil {
		c.RestaurantID = *u.RestaurantID
	}
	return s.tokens.Issue(c)
}
