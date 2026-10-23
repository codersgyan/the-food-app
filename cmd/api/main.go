// Command api runs the food delivery app's API server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/codersgyan/the-food-app/internal/auth"
	"github.com/codersgyan/the-food-app/internal/fake"
	"github.com/codersgyan/the-food-app/internal/handler"
	"github.com/codersgyan/the-food-app/internal/service"
	"github.com/codersgyan/the-food-app/internal/store"
)

// A login lasts 30 days. Tokens can't be cancelled early, so keep this short
// enough that a stolen phone's token doesn't work forever.
const tokenTTL = 30 * 24 * time.Hour

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Ctrl+C or a SIGTERM from Docker cancels ctx, and we shut down cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	tokens := auth.NewTokens(cfg.jwtSecret, tokenTTL)
	svc := service.New(service.Deps{
		Store:  store.New(pool),
		OTP:    &fake.OTP{Delay: cfg.otpDelay},
		Tokens: tokens,

		NearRadiusKm: cfg.nearRadiusKm,
	})

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler.New(svc, tokens),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	// Stop taking new requests, and give the ones in flight 10 seconds to finish.
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
