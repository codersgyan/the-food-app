package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/codersgyan/the-food-app/internal/fake"
)

// config is everything the server reads from the environment. Most settings
// have a default that works on your laptop. The two secrets don't: the code
// is public, so a default secret would be a secret everyone knows.
type config struct {
	addr         string
	databaseURL  string
	jwtSecret    string
	nearRadiusKm float64

	otpDelay fake.Delay

	// The payment provider signs its webhooks with this secret.
	paymentWebhookSecret string
	payDelay             fake.Delay
	payWebhookURL        string
	payFailRate          float64
	payDuplicateRate     float64
}

func loadConfig() (config, error) {
	cfg := config{
		addr:        env("ADDR", ":8080"),
		databaseURL: env("DATABASE_URL", "postgres://food:food@localhost:5440/food?sslmode=disable"),
		jwtSecret:   os.Getenv("JWT_SECRET"),

		paymentWebhookSecret: os.Getenv("PAYMENT_WEBHOOK_SECRET"),
		payWebhookURL:        env("FAKEPAY_WEBHOOK_URL", "http://localhost:8080/payments/webhook"),
	}
	for key, v := range map[string]string{"JWT_SECRET": cfg.jwtSecret, "PAYMENT_WEBHOOK_SECRET": cfg.paymentWebhookSecret} {
		if v == "" {
			return config{}, fmt.Errorf("%s is not set: on your laptop, run cp .env.example .env", key)
		}
	}
	var err error
	if cfg.otpDelay, err = fake.ParseDelay(env("FAKEOTP_DELAY", "200ms-500ms")); err != nil {
		return config{}, err
	}
	if cfg.payDelay, err = fake.ParseDelay(env("FAKEPAY_DELAY", "1s-2s")); err != nil {
		return config{}, err
	}
	if cfg.payFailRate, err = rate("FAKEPAY_FAIL_RATE"); err != nil {
		return config{}, err
	}
	if cfg.payDuplicateRate, err = rate("FAKEPAY_DUPLICATE_RATE"); err != nil {
		return config{}, err
	}
	if cfg.nearRadiusKm, err = strconv.ParseFloat(env("NEAR_RADIUS_KM", "7"), 64); err != nil {
		return config{}, fmt.Errorf("NEAR_RADIUS_KM: %w", err)
	}
	return cfg, nil
}

// rate reads a share from 0 to 1. It's 0 when the variable isn't set.
func rate(key string) (float64, error) {
	r, err := strconv.ParseFloat(env(key, "0"), 64)
	if err != nil || r < 0 || r > 1 {
		return 0, fmt.Errorf("%s must be a number from 0 to 1", key)
	}
	return r, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
