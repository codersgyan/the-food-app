package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/codersgyan/the-food-app/internal/fake"
)

// config is everything the server reads from the environment.
// The defaults are fine on your laptop. Never use them in production.
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
}

func loadConfig() (config, error) {
	cfg := config{
		addr:        env("ADDR", ":8080"),
		databaseURL: env("DATABASE_URL", "postgres://food:food@localhost:5440/food?sslmode=disable"),
		jwtSecret:   env("JWT_SECRET", "dev-only-jwt-secret"),

		paymentWebhookSecret: env("PAYMENT_WEBHOOK_SECRET", "dev-only-webhook-secret"),
		payWebhookURL:        env("FAKEPAY_WEBHOOK_URL", "http://localhost:8080/payments/webhook"),
	}
	var err error
	if cfg.otpDelay, err = fake.ParseDelay(env("FAKEOTP_DELAY", "200ms-500ms")); err != nil {
		return config{}, err
	}
	if cfg.payDelay, err = fake.ParseDelay(env("FAKEPAY_DELAY", "1s-2s")); err != nil {
		return config{}, err
	}
	if cfg.nearRadiusKm, err = strconv.ParseFloat(env("NEAR_RADIUS_KM", "7"), 64); err != nil {
		return config{}, fmt.Errorf("NEAR_RADIUS_KM: %w", err)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
