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

// mustLoadConfig reads the settings, and panics if one is missing or wrong:
// the server can't do anything useful without them.
func mustLoadConfig() config {
	return config{
		addr:         env("ADDR", ":8080"),
		databaseURL:  env("DATABASE_URL", "postgres://food:food@localhost:5440/food?sslmode=disable"),
		jwtSecret:    mustEnv("JWT_SECRET"),
		nearRadiusKm: mustFloat("NEAR_RADIUS_KM", "7"),

		otpDelay: mustDelay("FAKEOTP_DELAY", "200ms-500ms"),

		paymentWebhookSecret: mustEnv("PAYMENT_WEBHOOK_SECRET"),
		payDelay:             mustDelay("FAKEPAY_DELAY", "1s-2s"),
		payWebhookURL:        env("FAKEPAY_WEBHOOK_URL", "http://localhost:8080/payments/webhook"),
		payFailRate:          mustRate("FAKEPAY_FAIL_RATE"),
		payDuplicateRate:     mustRate("FAKEPAY_DUPLICATE_RATE"),
	}
}

// mustEnv reads a setting that has no default, like a secret.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(key + " is not set: on your laptop, run cp .env.example .env")
	}
	return v
}

func mustFloat(key, fallback string) float64 {
	f, err := strconv.ParseFloat(env(key, fallback), 64)
	if err != nil {
		panic(fmt.Sprintf("%s must be a number: %v", key, err))
	}
	return f
}

func mustDelay(key, fallback string) fake.Delay {
	d, err := fake.ParseDelay(env(key, fallback))
	if err != nil {
		panic(fmt.Sprintf("%s: %v", key, err))
	}
	return d
}

// mustRate reads a share from 0 to 1. It's 0 when the variable isn't set.
func mustRate(key string) float64 {
	r := mustFloat(key, "0")
	if r < 0 || r > 1 {
		panic(key + " must be a number from 0 to 1")
	}
	return r
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
