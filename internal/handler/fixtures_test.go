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
