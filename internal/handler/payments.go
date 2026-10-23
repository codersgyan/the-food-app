package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"io"
	"net/http"
)

// POST /v1/orders/{id}/pay starts the payment for an order. The answer comes
// back at once, with the order in "paying". The payment provider tells us
// later whether it went through, so the app checks GET /v1/orders/{id}.
func (h *handler) pay(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	paymentID, err := h.svc.Payments.Pay(r.Context(), claims(r).UserID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		OrderID   int64  `json:"order_id"`
		Status    string `json:"status"`
		PaymentID string `json:"payment_id"`
	}{id, "paying", paymentID})
}

// POST /payments/webhook is called by the payment provider, not by our app,
// to tell us how a payment went. It isn't under /v1, because its shape is set
// by the provider, not by our API's version.
func (h *handler) paymentWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"body too large"})
		return
	}
	// Anyone on the internet can call this URL, so check the request really
	// came from the provider: only the provider and we know the secret.
	if !validSignature(h.webhookSecret, body, r.Header.Get("X-Signature")) {
		writeJSON(w, http.StatusUnauthorized, errorBody{"bad signature"})
		return
	}

	var req struct {
		OrderID   int64  `json:"order_id"`
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"body is not valid JSON"})
		return
	}
	if err := h.svc.Payments.Settle(r.Context(), req.OrderID, req.PaymentID, req.Status); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"received": true})
}

// validSignature checks an HMAC-SHA256 of the body, in hex.
func validSignature(secret string, body []byte, signature string) bool {
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	// hmac.Equal takes the same time however many bytes match, so an
	// attacker can't guess the signature one byte at a time.
	return hmac.Equal(got, mac.Sum(nil))
}
