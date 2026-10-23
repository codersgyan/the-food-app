// Command migrate builds the database: "migrate up" applies every new migration,
// and "migrate down" rolls back the last one.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codersgyan/the-food-app/migrations"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		fmt.Fprintln(os.Stderr, "usage: migrate up|down")
		os.Exit(2)
	}

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://food:food@localhost:5440/food?sslmode=disable"
	}

	ctx := context.Background()
	var err error
	if os.Args[1] == "up" {
		err = migrations.Up(ctx, url)
	} else {
		err = migrations.Down(ctx, url)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}
