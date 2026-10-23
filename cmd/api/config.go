package main

import (
	"os"
)

// config is everything the server reads from the environment.
// The defaults are fine on your laptop. Never use them in production.
type config struct {
	addr        string
	databaseURL string
}

func loadConfig() config {
	return config{
		addr:        env("ADDR", ":8080"),
		databaseURL: env("DATABASE_URL", "postgres://food:food@localhost:5440/food?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
