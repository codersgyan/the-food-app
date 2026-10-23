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
	paid := a.placeOrder(k)
	a.setStatus(paid, "paid")
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
	if assert.Len(t, out.Orders, 1) {
		assert.Equal(t, paid, out.Orders[0].ID)
		assert.Len(t, out.Orders[0].Items, 2, "the kitchen needs to know what to cook")
	}

	res = a.do("GET", fmt.Sprintf("/v1/restaurants/%d/orders?status=paid", k.restaurant), otherStaff, nil)
	assert.Equal(t, http.StatusForbidden, res.status, "another restaurant's staff")
}
