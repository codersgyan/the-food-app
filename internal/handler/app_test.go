package handler_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/codersgyan/the-food-app/internal/fake"
	"github.com/codersgyan/the-food-app/internal/handler"
	"github.com/codersgyan/the-food-app/internal/service"
	"github.com/codersgyan/the-food-app/migrations"
)

// The tests run against a real Postgres, in its own database, so they check
// the real SQL. Start it with "make db".
var testDatabaseURL = "postgres://food:food@localhost:5440/food_test?sslmode=disable"

func TestMain(m *testing.M) {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		testDatabaseURL = url
	}
	if err := migrations.Up(context.Background(), testDatabaseURL); err != nil {
		fmt.Fprintln(os.Stderr, "migrate test database (is Postgres running? try make db):", err)
		os.Exit(1)
	}
	// Keep the test output readable: the server's request logs aren't needed here.
	slog.SetDefault(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}

// app is the whole server, running against an empty test database.
type app struct {
	t    *testing.T
	srv  *httptest.Server
	pool *pgxpool.Pool
}

func newApp(t *testing.T) *app {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// IDs keep counting up across tests, so nothing left over from one test
	// can point at a row in the next.
	_, err = pool.Exec(ctx, `TRUNCATE order_items, orders, dishes, addresses, users, restaurants`)
	require.NoError(t, err)

	svc := service.New(service.Deps{
		OTP: &fake.OTP{},
	})
	srv := httptest.NewServer(handler.New(svc))
	t.Cleanup(srv.Close)

	return &app{t: t, srv: srv, pool: pool}
}

type response struct {
	t      *testing.T
	status int
	body   []byte
}

// do sends a request with a JSON body (if body isn't nil) and the token (if it isn't empty).
func (a *app) do(method, path, token string, body any) response {
	a.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(a.t, err)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.srv.URL+path, r)
	require.NoError(a.t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := a.srv.Client().Do(req)
	require.NoError(a.t, err)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	require.NoError(a.t, err)
	return response{t: a.t, status: res.StatusCode, body: b}
}

// decode reads the response body into v.
func (r response) decode(v any) {
	r.t.Helper()
	require.NoError(r.t, json.Unmarshal(r.body, v), "body: %s", r.body)
}

// errorMessage returns the "error" field of an error response.
func (r response) errorMessage() string {
	var e struct {
		Error string `json:"error"`
	}
	r.decode(&e)
	return e.Error
}
