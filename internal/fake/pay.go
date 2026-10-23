package fake

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"log/slog"
	mrand "math/rand/v2"
	"net/http"
	"time"
)

// Payments works like a payment provider such as Razorpay. When we ask it to
// charge for an order, it answers at once with a payment ID. The customer
// then approves the payment in their UPI app, which takes a moment, and the
// provider tells our server how it went by calling our webhook.
//
// This fake approves payments after Delay and calls the webhook itself. By
// default every payment goes through, once. Real providers aren't that
// tidy, so FailRate and DuplicateRate let you make it behave like one.
type Payments struct {
	// Delay is how long the customer takes to approve the payment.
	Delay Delay
	// WebhookURL is our server's POST /payments/webhook.
	WebhookURL string
	// Secret signs every webhook, so our server can check it came from the provider.
	Secret string
	// FailRate is the share of payments that fail, from 0 (none) to 1 (all),
	// like a customer typing the wrong PIN.
	FailRate float64
	// DuplicateRate is the share of webhooks sent twice. Real providers
	// promise to send each webhook at least once, not exactly once.
	DuplicateRate float64
	Client        *http.Client
}

// Webhook is what the provider sends to our server. The requirements lesson
// shows only order_id and status. Real providers also send their payment ID,
// and our server saves it in the order's payment_id column.
type Webhook struct {
	OrderID   int64  `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"` // "captured" or "failed"
}

// Charge starts a payment for an order and returns its payment ID. The result
// arrives later, at the webhook.
func (p *Payments) Charge(ctx context.Context, orderID int64, amount int) (string, error) {
	id := "pay_" + rand.Text()[:14]
	slog.InfoContext(ctx, "fake payment started", "order_id", orderID, "payment_id", id, "amount", amount)
	go p.approve(orderID, id)
	return id, nil
}

func (p *Payments) approve(orderID int64, paymentID string) {
	ctx := context.Background()
	if err := p.Delay.wait(ctx); err != nil {
		return
	}
	w := Webhook{OrderID: orderID, PaymentID: paymentID, Status: "captured"}
	if mrand.Float64() < p.FailRate {
		w.Status = "failed"
	}
	p.send(ctx, w)
	if mrand.Float64() < p.DuplicateRate {
		p.send(ctx, w)
	}
}

func (p *Payments) send(ctx context.Context, w Webhook) {
	body, err := json.Marshal(w)
	if err != nil {
		slog.Error("fake payment webhook", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.WebhookURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("fake payment webhook", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", Sign(p.Secret, body))

	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		slog.Error("fake payment webhook", "order_id", w.OrderID, "err", err)
		return
	}
	res.Body.Close()
	if res.StatusCode >= 300 {
		slog.Error("fake payment webhook", "order_id", w.OrderID, "status", res.StatusCode)
	}
}

// Sign returns the signature of a webhook body: an HMAC-SHA256 of the body,
// keyed with the secret we share with the provider, in hex. Razorpay signs
// its webhooks the same way.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
