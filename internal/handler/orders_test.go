package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kitchen is one restaurant with two dishes, and a customer with a saved address.
type kitchen struct {
	restaurant, biryani, raita int64
	customer                   string
	customerID, address        int64
}

func (a *app) kitchen() kitchen {
	a.t.Helper()
	k := kitchen{restaurant: a.restaurant("Biryani House", 12.9352, 77.6245)}
	k.biryani = a.dish(k.restaurant, "Chicken biryani", 25000)
	k.raita = a.dish(k.restaurant, "Raita", 4000)
	k.customer, k.customerID = a.customer("+919800000001")
	k.address = a.address(k.customerID, "Flat 302, Koramangala")
	return k
}

type item struct {
	DishID   int64 `json:"dish_id"`
	Quantity int   `json:"quantity"`
	Price    int   `json:"price,omitzero"` // only to show the server ignores it
}

type newOrder struct {
	RestaurantID int64  `json:"restaurant_id"`
	AddressID    int64  `json:"address_id"`
	Items        []item `json:"items"`
}

// placeOrder places 2 biryanis and 1 raita (₹540) and returns the order ID.
func (a *app) placeOrder(k kitchen) int64 {
	a.t.Helper()
	res := a.do("POST", "/v1/orders", k.customer, newOrder{k.restaurant, k.address, []item{
		{DishID: k.biryani, Quantity: 2}, {DishID: k.raita, Quantity: 1},
	}})
	require.Equal(a.t, http.StatusCreated, res.status, "body: %s", res.body)
	var o struct {
		ID int64 `json:"id"`
	}
	res.decode(&o)
	return o.ID
}

func (a *app) orderCount() int {
	a.t.Helper()
	var n int
	require.NoError(a.t, a.pool.QueryRow(a.t.Context(), `SELECT count(*) FROM orders`).Scan(&n))
	return n
}

func TestPlaceOrderTakesPricesFromTheMenuNotTheApp(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()

	res := a.do("POST", "/v1/orders", k.customer, newOrder{k.restaurant, k.address, []item{
		{DishID: k.biryani, Quantity: 2, Price: 1}, // the app claims biryani costs 1 paisa
		{DishID: k.raita, Quantity: 1},
	}})

	require.Equal(t, http.StatusCreated, res.status, "body: %s", res.body)
	var o struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Total  int    `json:"total"`
	}
	res.decode(&o)
	assert.Equal(t, "pending", o.Status)
	assert.Equal(t, 2*25000+4000, o.Total)

	var biryaniPrice int
	var address string
	require.NoError(t, a.pool.QueryRow(t.Context(), `
		SELECT oi.price, o.delivery_address
		FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE oi.order_id = $1 AND oi.dish_id = $2`, o.ID, k.biryani,
	).Scan(&biryaniPrice, &address))
	assert.Equal(t, 25000, biryaniPrice, "the price is copied into the order")
	assert.Equal(t, "Flat 302, Koramangala", address, "the address is copied into the order")
}

func TestPlaceOrderRefusesASoldOutDishAndSavesNothing(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	a.soldOut(k.raita)

	res := a.do("POST", "/v1/orders", k.customer, newOrder{k.restaurant, k.address, []item{
		{DishID: k.biryani, Quantity: 2}, {DishID: k.raita, Quantity: 1},
	}})

	assert.Equal(t, http.StatusConflict, res.status)
	assert.Equal(t, "Raita is sold out", res.errorMessage())
	assert.Equal(t, 0, a.orderCount())
}

func TestPlaceOrderRefusesDishesAndAddressesThatDontBelong(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	dosa := a.dish(a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412), "Masala dosa", 9000)
	_, vikramID := a.customer("+919800000002")
	vikramsAddress := a.address(vikramID, "12, Indiranagar")

	res := a.do("POST", "/v1/orders", k.customer, newOrder{k.restaurant, k.address, []item{{DishID: dosa, Quantity: 1}}})
	assert.Equal(t, http.StatusBadRequest, res.status, "a dish from another restaurant")

	res = a.do("POST", "/v1/orders", k.customer, newOrder{k.restaurant, vikramsAddress, []item{{DishID: k.biryani, Quantity: 1}}})
	assert.Equal(t, http.StatusBadRequest, res.status, "someone else's address")

	assert.Equal(t, 0, a.orderCount())
}
