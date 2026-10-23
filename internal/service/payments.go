package service

import (
	"context"
	"log/slog"

	"github.com/codersgyan/the-food-app/internal/store"
)

// PaymentProvider charges the customer. It answers at once with a payment
// ID, and tells us later, through our webhook, whether the payment went through.
type PaymentProvider interface {
	Charge(ctx context.Context, orderID int64, amount int) (paymentID string, err error)
}

// PaymentService takes payment for orders. Its one rule: never charge a
// customer twice for one order.
type PaymentService struct {
	store    *store.Store
	payments PaymentProvider
	orders   *OrderService
}

// Pay starts the payment for one of the user's orders and returns its payment ID.
func (s *PaymentService) Pay(ctx context.Context, userID, orderID int64) (string, error) {
	o, err := s.orders.Get(ctx, userID, orderID)
	if err != nil {
		return "", err
	}

	// Only one request can move the order from pending to paying. If the
	// customer taps Pay twice, the second tap finds it already paying and
	// stops here, before it charges anyone.
	moved, err := s.store.SetStatus(ctx, orderID, "pending", "paying")
	if err != nil {
		return "", err
	}
	if !moved {
		return "", fail(ErrConflict, "order %d is not waiting for payment", orderID)
	}

	paymentID, err := s.payments.Charge(ctx, orderID, o.Total)
	if err != nil {
		// The charge never started, so let the customer try again.
		if _, rerr := s.store.SetStatus(context.WithoutCancel(ctx), orderID, "paying", "pending"); rerr != nil {
			return "", rerr
		}
		return "", err
	}
	if err := s.store.SetPaymentID(ctx, orderID, paymentID); err != nil {
		return "", err
	}
	return paymentID, nil
}

// Settle records what the payment provider says about an order's payment:
// "captured" means the money is ours, and "failed" means the customer can
// try again. Providers send the same webhook more than once, so a webhook
// for an order that's no longer paying changes nothing.
func (s *PaymentService) Settle(ctx context.Context, orderID int64, paymentID, status string) error {
	var moved bool
	var err error
	switch status {
	case "captured":
		moved, err = s.store.MarkPaid(ctx, orderID, paymentID)
	case "failed":
		moved, err = s.store.SetStatus(ctx, orderID, "paying", "pending")
	default:
		return fail(ErrInvalid, "unknown payment status %q", status)
	}
	if err != nil {
		return err
	}
	if !moved {
		slog.InfoContext(ctx, "payment webhook changed nothing", "order_id", orderID, "payment_id", paymentID, "status", status)
	}
	return nil
}
