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

// Order returns an order with its items.
func (s *Store) Order(ctx context.Context, id int64) (Order, error) {
	var o Order
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, restaurant_id, placed_at, delivery_address, total, status, payment_id
		FROM orders
		WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.UserID, &o.RestaurantID, &o.PlacedAt, &o.DeliveryAddress, &o.Total, &o.Status, &o.PaymentID)
	if err != nil {
		return Order{}, notFound(err)
	}
	o.Items, err = s.orderItems(ctx, id)
	return o, err
}

// orderItems uses the index on order_items.order_id.
func (s *Store) orderItems(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT oi.dish_id, d.name, oi.quantity, oi.price
		FROM order_items oi
		JOIN dishes d ON d.id = oi.dish_id
		WHERE oi.order_id = $1
		ORDER BY oi.dish_id`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.DishID, &it.Name, &it.Quantity, &it.Price); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// SetStatus moves an order from one status to another, and reports whether it
// did. The WHERE status = from is the guard: if two requests try the same
// move at the same moment, only one of them finds the order still in from.
func (s *Store) SetStatus(ctx context.Context, id int64, from, to string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE orders SET status = $3, updated_at = now()
		WHERE id = $1 AND status = $2`,
		id, from, to,
	)
	return tag.RowsAffected() == 1, err
}

// SetPaymentID saves the payment provider's ID for the order's payment.
func (s *Store) SetPaymentID(ctx context.Context, id int64, paymentID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE orders SET payment_id = $2, updated_at = now()
		WHERE id = $1`,
		id, paymentID,
	)
	return err
}

// MarkPaid moves a paying order to paid and saves its payment ID. Like
// SetStatus, it reports whether it moved the order.
func (s *Store) MarkPaid(ctx context.Context, id int64, paymentID string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE orders SET status = 'paid', payment_id = $2, updated_at = now()
		WHERE id = $1 AND status = 'paying'`,
		id, paymentID,
	)
	return tag.RowsAffected() == 1, err
}
