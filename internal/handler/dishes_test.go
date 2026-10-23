package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPriceDoesNotChangeAnOrderAlreadyPlaced(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	staff := a.staffToken(k)
	id := a.placeOrder(k) // 2 biryanis at ₹250 and a raita at ₹40

	res := a.do("PATCH", fmt.Sprintf("/v1/dishes/%d", k.biryani), staff, map[string]int{"price": 30000})
	require.Equal(t, http.StatusOK, res.status, "body: %s", res.body)

	res = a.do("GET", fmt.Sprintf("/v1/orders/%d", id), k.customer, nil)
	var o struct {
		Total int `json:"total"`
		Items []struct {
			Price int `json:"price"`
		} `json:"items"`
	}
	res.decode(&o)
	assert.Equal(t, 2*25000+4000, o.Total, "the bill never changes after it's placed")
	require.Len(t, o.Items, 2)
	assert.Equal(t, 25000, o.Items[0].Price)
}

func TestStaffChangeOnlyTheirOwnDishes(t *testing.T) {
	a := newApp(t)
	k := a.kitchen()
	dosa := a.dish(a.restaurant("Udupi Tiffin Room", 12.9719, 77.6412), "Masala dosa", 9000)
	staff := a.staffToken(k)

	res := a.do("PATCH", fmt.Sprintf("/v1/dishes/%d", dosa), staff, map[string]bool{"sold_out": true})
	assert.Equal(t, http.StatusNotFound, res.status, "another restaurant's dish")

	res = a.do("PATCH", fmt.Sprintf("/v1/dishes/%d", k.raita), staff, map[string]bool{"sold_out": true})
	assert.Equal(t, http.StatusOK, res.status)
	var d struct {
		Price   int  `json:"price"`
		SoldOut bool `json:"sold_out"`
	}
	res.decode(&d)
	assert.True(t, d.SoldOut)
	assert.Equal(t, 4000, d.Price, "a field that wasn't sent stays as it was")
}
