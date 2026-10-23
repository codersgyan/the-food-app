package service

import (
	"context"
	"regexp"
)

// OTPSender texts a one-time login code to a phone, and checks the code the
// user types. The OTP service stores the codes, so we don't.
type OTPSender interface {
	Send(ctx context.Context, phone string) error
	Check(ctx context.Context, phone, code string) (bool, error)
}

// AuthService logs users in with their phone number and a code texted to it.
type AuthService struct {
	otp OTPSender
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
