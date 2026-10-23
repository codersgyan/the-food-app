package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomerRoutesNeedACustomerToken(t *testing.T) {
	a := newApp(t)
	a.staff("+919810000001", a.restaurant("Biryani House", 12.9352, 77.6245))
	staffToken := a.login("+919810000001")

	assert.Equal(t, http.StatusUnauthorized, a.do("GET", "/v1/me", "", nil).status, "no token")
	assert.Equal(t, http.StatusUnauthorized, a.do("GET", "/v1/me", "not-a-token", nil).status, "bad token")
	assert.Equal(t, http.StatusForbidden, a.do("GET", "/v1/me", staffToken, nil).status, "restaurant token")
}

func TestMeShowsOnlyYourOwnAddresses(t *testing.T) {
	a := newApp(t)
	asha, ashaID := a.customer("+919800000001")
	_, vikramID := a.customer("+919800000002")
	a.address(ashaID, "Flat 302, Koramangala")
	a.address(vikramID, "12, Indiranagar")

	res := a.do("GET", "/v1/me", asha, nil)

	assert.Equal(t, http.StatusOK, res.status)
	var me struct {
		Name      *string `json:"name"`
		Addresses []struct {
			Address string `json:"address"`
		} `json:"addresses"`
	}
	res.decode(&me)
	assert.Nil(t, me.Name, "no request sets a name yet")
	assert.Len(t, me.Addresses, 1)
	assert.Equal(t, "Flat 302, Koramangala", me.Addresses[0].Address)
}
