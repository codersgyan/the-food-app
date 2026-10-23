// Package migrations holds the SQL files that build the database, one change per file.
package migrations

import (
	"context"
	"database/sql"
	"embed"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver for database/sql
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// Up applies every migration that hasn't run yet.
func Up(ctx context.Context, databaseURL string) error {
	return run(ctx, databaseURL, func(p *goose.Provider) error {
		_, err := p.Up(ctx)
		return err
	})
}

// Down rolls back the most recent migration.
func Down(ctx context.Context, databaseURL string) error {
	return run(ctx, databaseURL, func(p *goose.Provider) error {
		_, err := p.Down(ctx)
		return err
	})
}

func run(ctx context.Context, databaseURL string, fn func(*goose.Provider) error) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	p, err := goose.NewProvider(goose.DialectPostgres, db, files)
	if err != nil {
		return err
	}
	return fn(p)
}
