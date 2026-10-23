package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// staffToken logs in a staff member of the kitchen's restaurant.
func (a *app) staffToken(k kitchen) string {
	a.t.Helper()
	a.staff("+919810000001", k.restaurant)
	return a.login("+919810000001")
}

func TestKitchenSeesOnlyItsOwnPaidOrders(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	staff := a.staffToken(k)
	first := a.placeOrder(k)
	second := a.placeOrder(k)
	a.setStatus(first, "paid")
	a.setStatus(second, "paid")
	a.placeOrder(k) // still pending
	other := a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412)
	a.staff("+919810000002", other)
	otherStaff := a.login("+919810000002")

	res := a.do("GET", fmt.Sprintf("/v1/restaurants/%d/orders?status=paid", k.restaurant), staff, nil)

	assert.Equal(t, http.StatusOK, res.status)
	var out struct {
		Orders []struct {
			ID    int64 `json:"id"`
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"orders"`
	}
	res.decode(&out)
	if assert.Len(t, out.Orders, 2) {
		assert.Equal(t, first, out.Orders[0].ID, "oldest first")
		assert.Equal(t, second, out.Orders[1].ID)
		// Each order has its own biryani and raita, so the kitchen knows what to cook for which order.
		assert.Len(t, out.Orders[0].Items, 2)
		assert.Len(t, out.Orders[1].Items, 2)
	}

	res = a.do("GET", fmt.Sprintf("/v1/restaurants/%d/orders?status=paid", k.restaurant), otherStaff, nil)
	assert.Equal(t, http.StatusForbidden, res.status, "another restaurant's staff")
}

func TestKitchenAcceptsOnlyItsOwnPaidOrders(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	staff := a.staffToken(k)
	id := a.placeOrder(k)
	accept := fmt.Sprintf("/v1/orders/%d/accept", id)

	assert.Equal(t, http.StatusConflict, a.do("POST", accept, staff, nil).status, "not paid yet")

	a.setStatus(id, "paid")
	a.staff("+919810000002", a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412))
	assert.Equal(t, http.StatusNotFound, a.do("POST", accept, a.login("+919810000002"), nil).status, "another restaurant")

	assert.Equal(t, http.StatusOK, a.do("POST", accept, staff, nil).status)
	assert.Equal(t, "accepted", a.orderStatus(k.customer, id))
}

func TestOnlyAnAcceptedOrderCanBeReady(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	staff := a.staffToken(k)
	id := a.placeOrder(k)
	a.setStatus(id, "paid")
	ready := fmt.Sprintf("/v1/orders/%d/ready", id)

	assert.Equal(t, http.StatusConflict, a.do("POST", ready, staff, nil).status, "not accepted yet")

	a.setStatus(id, "accepted")
	assert.Equal(t, http.StatusOK, a.do("POST", ready, staff, nil).status)
	assert.Equal(t, "ready", a.orderStatus(k.customer, id))
}
