package fake

import (
	"context"
	"log/slog"
)

// OTPCode is the only code the fake OTP service accepts.
const OTPCode = "123456"

// OTP works like an OTP service such as Twilio Verify: it sends a code to a
// phone, then checks the code the user types. This fake sends nothing. It
// logs the code, and the code is always OTPCode.
type OTP struct {
	Delay Delay
}

// Send pretends to text a login code to phone.
func (o *OTP) Send(ctx context.Context, phone string) error {
	if err := o.Delay.wait(ctx); err != nil {
		return err
	}
	slog.InfoContext(ctx, "fake OTP sent", "phone", phone, "code", OTPCode)
	return nil
}

// Check reports whether code is the right code for phone.
func (o *OTP) Check(ctx context.Context, phone, code string) (bool, error) {
	if err := o.Delay.wait(ctx); err != nil {
		return false, err
	}
	return code == OTPCode, nil
}
