// Package auth issues and checks the token a user gets when they log in.
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// The two kinds of user. Restaurant staff use the restaurant's tablet.
const (
	RoleCustomer   = "customer"
	RoleRestaurant = "restaurant"
)

// ErrInvalidToken means the token is malformed, signed with another key, or expired.
var ErrInvalidToken = errors.New("invalid token")

// Claims is what a token says about its user.
type Claims struct {
	UserID int64
	Role   string
	// RestaurantID is the restaurant a staff member works for. It's 0 for customers.
	RestaurantID int64
}

// Tokens issues and checks JWTs signed with one secret key. Nothing is stored:
// the signature alone proves we issued the token. The cost is that we can't
// cancel a token before it expires.
type Tokens struct {
	secret []byte
	ttl    time.Duration
}

func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl}
}

type jwtClaims struct {
	Role         string `json:"role"`
	RestaurantID int64  `json:"restaurant_id,omitempty"`
	jwt.RegisteredClaims
}

// Issue returns a signed token for c that expires after the Tokens' ttl.
func (t *Tokens) Issue(c Claims) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		Role:         c.Role,
		RestaurantID: c.RestaurantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(c.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	})
	return token.SignedString(t.secret)
}

// Parse checks the token's signature and expiry, and returns its claims.
func (t *Tokens) Parse(s string) (Claims, error) {
	var jc jwtClaims
	_, err := jwt.ParseWithClaims(s, &jc,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	id, err := strconv.ParseInt(jc.Subject, 10, 64)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: id, Role: jc.Role, RestaurantID: jc.RestaurantID}, nil
}
