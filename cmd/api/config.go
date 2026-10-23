package main

import (
	"os"

	"github.com/codersgyan/the-food-app/internal/fake"
)

// config is everything the server reads from the environment.
// The defaults are fine on your laptop. Never use them in production.
type config struct {
	addr        string
	databaseURL string
	jwtSecret   string
	otpDelay    fake.Delay
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
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
