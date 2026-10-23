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
	otpDelay     fake.Delay
	nearRadiusKm float64
}

func loadConfig() (config, error) {
	cfg := config{
		addr:        env("ADDR", ":8080"),
		databaseURL: env("DATABASE_URL", "postgres://food:food@localhost:5440/food?sslmode=disable"),
		jwtSecret:   env("JWT_SECRET", "dev-only-jwt-secret"),
	}
	var err error
	if cfg.otpDelay, err = fake.ParseDelay(env("FAKEOTP_DELAY", "200ms-500ms")); err != nil {
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
