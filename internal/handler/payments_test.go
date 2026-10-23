package handler_test

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codersgyan/the-food-app/internal/fake"
)

func TestTappingPayManyTimesAtOnceChargesOnce(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)

	// Ten taps on Pay, all arriving at the same moment.
	statuses := make([]int, 10)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), k.customer, nil).status
		})
	}
	wg.Wait()

	assert.Equal(t, 1, count(statuses, http.StatusOK))
	assert.Equal(t, 9, count(statuses, http.StatusConflict))
	assert.Equal(t, int32(1), a.payments.charges.Load())
}

func TestCustomerCannotPayForSomeoneElsesOrder(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)
	vikram, _ := a.customer("+919800000002")

	res := a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), vikram, nil)

	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, int32(0), a.payments.charges.Load())
}

func count(list []int, v int) int {
	n := 0
	for _, x := range list {
		if x == v {
			n++
		}
	}
	return n
}

// webhook sends a payment webhook the way the provider does, signed with secret.
func (a *app) webhook(secret string, w fake.Webhook) response {
	a.t.Helper()
	body, err := json.Marshal(w)
	require.NoError(a.t, err)
	req, err := http.NewRequest("POST", a.srv.URL+"/payments/webhook", bytes.NewReader(body))
	require.NoError(a.t, err)
	req.Header.Set("X-Signature", fake.Sign(secret, body))
	res, err := a.srv.Client().Do(req)
	require.NoError(a.t, err)
	res.Body.Close()
	return response{t: a.t, status: res.StatusCode}
}

func TestPaidOnceTheProviderApproves(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)

	res := a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), k.customer, nil)
	require.Equal(t, http.StatusOK, res.status, "body: %s", res.body)

	// The fake provider approves the payment and calls our webhook a moment later.
	assert.Eventually(t, func() bool { return a.orderStatus(k.customer, id) == "paid" },
		2*time.Second, 20*time.Millisecond)
}

func TestWebhookWithABadSignatureChangesNothing(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)
	a.setStatus(id, "paying")

	res := a.webhook("not-the-secret", fake.Webhook{OrderID: id, PaymentID: "pay_forged", Status: "captured"})

	assert.Equal(t, http.StatusUnauthorized, res.status)
	assert.Equal(t, "paying", a.orderStatus(k.customer, id))
}

func TestLateDuplicateWebhookDoesNotMoveTheOrderBack(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)
	a.setStatus(id, "paying")
	captured := fake.Webhook{OrderID: id, PaymentID: "pay_29QQoUBi66xm2f", Status: "captured"}

	assert.Equal(t, http.StatusOK, a.webhook(webhookSecret, captured).status)
	a.setStatus(id, "accepted") // the kitchen has started cooking
	// The provider sends the same webhook again, and our server must still answer 200,
	// or the provider keeps retrying.
	assert.Equal(t, http.StatusOK, a.webhook(webhookSecret, captured).status)

	assert.Equal(t, "accepted", a.orderStatus(k.customer, id))
}

func TestFailedPaymentLetsTheCustomerPayAgain(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	id := a.placeOrder(k)
	a.setStatus(id, "paying")

	res := a.webhook(webhookSecret, fake.Webhook{OrderID: id, PaymentID: "pay_declined", Status: "failed"})
	require.Equal(t, http.StatusOK, res.status)
	require.Equal(t, "pending", a.orderStatus(k.customer, id))

	res = a.do("POST", fmt.Sprintf("/v1/orders/%d/pay", id), k.customer, nil)
	assert.Equal(t, http.StatusOK, res.status)
	assert.Eventually(t, func() bool { return a.orderStatus(k.customer, id) == "paid" },
		2*time.Second, 20*time.Millisecond)
}
