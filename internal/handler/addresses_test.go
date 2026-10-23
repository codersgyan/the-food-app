package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddedAddressIsSavedForThatCustomer(t *testing.T) {
	a := newApp(t)
	token, _ := a.customer("+919800000001")

	res := a.do("POST", "/v1/addresses", token, map[string]string{"address": "  Flat 302, Koramangala  "})
	require.Equal(t, http.StatusCreated, res.status)
	var created struct {
		ID int64 `json:"id"`
	}
	res.decode(&created)

	var me struct {
		Addresses []struct {
			ID      int64  `json:"id"`
			Address string `json:"address"`
		} `json:"addresses"`
	}
	a.do("GET", "/v1/me", token, nil).decode(&me)
	require.Len(t, me.Addresses, 1)
	assert.Equal(t, created.ID, me.Addresses[0].ID)
	assert.Equal(t, "Flat 302, Koramangala", me.Addresses[0].Address)

	res = a.do("POST", "/v1/addresses", token, map[string]string{"address": "   "})
	assert.Equal(t, http.StatusBadRequest, res.status, "blank address")
}
