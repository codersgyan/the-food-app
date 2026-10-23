package service

// Deps is everything the services need from outside.
type Deps struct {
	OTP OTPSender
}

// Services holds one service for each part of the app.
type Services struct {
	Auth *AuthService
}

func New(d Deps) *Services {
	return &Services{
		Auth: &AuthService{otp: d.OTP},
	}
}
