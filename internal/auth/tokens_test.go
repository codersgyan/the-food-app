package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codersgyan/the-food-app/internal/auth"
)

func TestParseRejectsTokensWeDidNotIssueOrThatExpired(t *testing.T) {
	ours := auth.NewTokens("our-secret", time.Hour)
	claims := auth.Claims{UserID: 17, Role: auth.RoleCustomer}

	other, err := auth.NewTokens("someone-elses-secret", time.Hour).Issue(claims)
	require.NoError(t, err)
	_, err = ours.Parse(other)
	assert.ErrorIs(t, err, auth.ErrInvalidToken, "signed with another key")

	expired, err := auth.NewTokens("our-secret", -time.Minute).Issue(claims)
	require.NoError(t, err)
	_, err = ours.Parse(expired)
	assert.ErrorIs(t, err, auth.ErrInvalidToken, "expired")
}
