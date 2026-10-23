package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
