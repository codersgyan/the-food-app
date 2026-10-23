package store

import (
	"context"
	"time"
)

type Order struct {
	ID              int64
	UserID          int64
	RestaurantID    int64
	PlacedAt        time.Time
	DeliveryAddress string
	Total           int // in paise
	Status          string
	PaymentID       *string
	Items           []OrderItem
}

type OrderItem struct {
	DishID   int64
	Name     string
	Quantity int
	Price    int // in paise, for one, when the order was placed
}

// InsertOrder saves a new order and its items, and returns it with its ID,
// placed_at and status filled in. Call it inside InTx, so the order and its
// items are saved together.
func (s *Store) InsertOrder(ctx context.Context, o Order) (Order, error) {
	err := s.db.QueryRow(ctx, `
		INSERT INTO orders (user_id, restaurant_id, delivery_address, total)
		VALUES ($1, $2, $3, $4)
		RETURNING id, placed_at, status`,
		o.UserID, o.RestaurantID, o.DeliveryAddress, o.Total,
	).Scan(&o.ID, &o.PlacedAt, &o.Status)
	if err != nil {
		return Order{}, err
	}

	dishIDs := make([]int64, len(o.Items))
	quantities := make([]int, len(o.Items))
	prices := make([]int, len(o.Items))
	for i, it := range o.Items {
		dishIDs[i], quantities[i], prices[i] = it.DishID, it.Quantity, it.Price
	}
	// unnest turns the three lists into rows, so all the items go in with one statement.
	_, err = s.db.Exec(ctx, `
		INSERT INTO order_items (order_id, dish_id, quantity, price)
		SELECT $1, * FROM unnest($2::bigint[], $3::int[], $4::int[])`,
		o.ID, dishIDs, quantities, prices,
	)
	if err != nil {
		return Order{}, err
	}
	return o, nil
}
