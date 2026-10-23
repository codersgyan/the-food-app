package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codersgyan/the-food-app/internal/auth"
)

func TestLoginRejectsPhoneNotInInternationalFormat(t *testing.T) {
	a := newApp(t)

	// Every code we text costs money, so a malformed number never reaches the OTP service.
	for _, phone := range []string{"9845012345", "+91 98450 12345", ""} {
		res := a.do("POST", "/v1/login", "", map[string]string{"phone": phone})
		assert.Equal(t, http.StatusBadRequest, res.status, "phone %q", phone)
	}

	res := a.do("POST", "/v1/login", "", map[string]string{"phone": "+919845012345"})
	assert.Equal(t, http.StatusOK, res.status)
}

func TestVerifyLoginRejectsWrongCode(t *testing.T) {
	a := newApp(t)

	res := a.do("POST", "/v1/login/verify", "", map[string]string{"phone": "+919845012345", "code": "000000"})
	assert.Equal(t, http.StatusUnauthorized, res.status)
}

func TestFirstLoginCreatesOneCustomerAndLaterLoginsReuseIt(t *testing.T) {
	a := newApp(t)
	phone := "+919845012345"

	first, err := a.tokens.Parse(a.login(phone))
	require.NoError(t, err)
	second, err := a.tokens.Parse(a.login(phone))
	require.NoError(t, err)

	assert.Equal(t, first.UserID, second.UserID)
	assert.Equal(t, auth.RoleCustomer, first.Role)
	var users int
	require.NoError(t, a.pool.QueryRow(t.Context(), `SELECT count(*) FROM users WHERE phone = $1`, phone).Scan(&users))
	assert.Equal(t, 1, users)
}

func TestRestaurantStaffLoginKeepsTheirRestaurant(t *testing.T) {
	a := newApp(t)
	restaurantID := a.restaurant("Biryani House", 12.9352, 77.6245)
	a.staff("+919810000001", restaurantID)

	claims, err := a.tokens.Parse(a.login("+919810000001"))
	require.NoError(t, err)

	assert.Equal(t, auth.RoleRestaurant, claims.Role)
	assert.Equal(t, restaurantID, claims.RestaurantID)
}
