package handler_test

import "github.com/stretchr/testify/require"

// The helpers below put rows straight into the database, for the things the
// API has no request for, like adding a restaurant.

func (a *app) restaurant(name string, lat, lng float64) int64 {
	a.t.Helper()
	var id int64
	err := a.pool.QueryRow(a.t.Context(),
		`INSERT INTO restaurants (name, address, lat, lng) VALUES ($1, 'Somewhere in Bengaluru', $2, $3) RETURNING id`,
		name, lat, lng,
	).Scan(&id)
	require.NoError(a.t, err)
	return id
}

func (a *app) staff(phone string, restaurantID int64) {
	a.t.Helper()
	_, err := a.pool.Exec(a.t.Context(),
		`INSERT INTO users (phone, role, restaurant_id) VALUES ($1, 'restaurant', $2)`,
		phone, restaurantID,
	)
	require.NoError(a.t, err)
}

func (a *app) address(userID int64, address string) int64 {
	a.t.Helper()
	var id int64
	err := a.pool.QueryRow(a.t.Context(),
		`INSERT INTO addresses (user_id, address) VALUES ($1, $2) RETURNING id`, userID, address,
	).Scan(&id)
	require.NoError(a.t, err)
	return id
}

// dish adds a dish that costs price paise.
func (a *app) dish(restaurantID int64, name string, price int) int64 {
	a.t.Helper()
	var id int64
	err := a.pool.QueryRow(a.t.Context(),
		`INSERT INTO dishes (restaurant_id, name, price) VALUES ($1, $2, $3) RETURNING id`,
		restaurantID, name, price,
	).Scan(&id)
	require.NoError(a.t, err)
	return id
}

func (a *app) soldOut(dishID int64) {
	a.t.Helper()
	_, err := a.pool.Exec(a.t.Context(), `UPDATE dishes SET sold_out = true WHERE id = $1`, dishID)
	require.NoError(a.t, err)
}

// setStatus puts an order straight into a status, without going through the API.
func (a *app) setStatus(orderID int64, status string) {
	a.t.Helper()
	_, err := a.pool.Exec(a.t.Context(), `UPDATE orders SET status = $2 WHERE id = $1`, orderID, status)
	require.NoError(a.t, err)
}
