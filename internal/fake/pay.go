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
	"net/http"
	"time"
)

// Payments works like a payment provider such as Razorpay. When we ask it to
// charge for an order, it answers at once with a payment ID. The customer
// then approves the payment in their UPI app, which takes a moment, and the
// provider tells our server how it went by calling our webhook.
//
// This fake approves every payment after Delay, as if every customer typed
// the right UPI PIN, and calls the webhook itself.
type Payments struct {
	// Delay is how long the customer takes to approve the payment.
	Delay Delay
	// WebhookURL is our server's POST /payments/webhook.
	WebhookURL string
	// Secret signs every webhook, so our server can check it came from the provider.
	Secret string
	Client *http.Client
}

// Webhook is what the provider sends to our server.
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
	p.send(ctx, Webhook{OrderID: orderID, PaymentID: paymentID, Status: "captured"})
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
