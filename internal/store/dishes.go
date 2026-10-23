package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Dish struct {
	ID           int64
	RestaurantID int64
	Name         string
	Price        int // in paise
	SoldOut      bool
}

// DishesOf returns a restaurant's dishes. It uses the index on restaurant_id.
func (s *Store) DishesOf(ctx context.Context, restaurantID int64) ([]Dish, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, restaurant_id, name, price, sold_out
		FROM dishes
		WHERE restaurant_id = $1
		ORDER BY id`,
		restaurantID,
	)
	if err != nil {
		return nil, err
	}
	return scanDishes(rows)
}

func scanDishes(rows pgx.Rows) ([]Dish, error) {
	defer rows.Close()
	var list []Dish
	for rows.Next() {
		var d Dish
		if err := rows.Scan(&d.ID, &d.RestaurantID, &d.Name, &d.Price, &d.SoldOut); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// RestaurantDishes returns the dishes with these IDs that belong to the restaurant.
func (s *Store) RestaurantDishes(ctx context.Context, restaurantID int64, ids []int64) ([]Dish, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, restaurant_id, name, price, sold_out
		FROM dishes
		WHERE restaurant_id = $1 AND id = ANY($2)`,
		restaurantID, ids,
	)
	if err != nil {
		return nil, err
	}
	return scanDishes(rows)
}

// UpdateDish changes a dish's price, sold_out, or both (nil means "leave it
// as it is"), but only if the dish belongs to the restaurant.
func (s *Store) UpdateDish(ctx context.Context, restaurantID, id int64, price *int, soldOut *bool) (Dish, error) {
	var d Dish
	err := s.db.QueryRow(ctx, `
		UPDATE dishes
		SET price = COALESCE($3, price),
		    sold_out = COALESCE($4, sold_out),
		    updated_at = now()
		WHERE id = $1 AND restaurant_id = $2
		RETURNING id, restaurant_id, name, price, sold_out`,
		id, restaurantID, price, soldOut,
	).Scan(&d.ID, &d.RestaurantID, &d.Name, &d.Price, &d.SoldOut)
	return d, notFound(err)
}
