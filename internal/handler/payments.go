package handler

import "net/http"

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
