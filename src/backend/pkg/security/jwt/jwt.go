package security_jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager: struct to hold the JWT manager.
type JWTManager struct {
	secretKey []byte
	Expiry    time.Duration
}

// NewJWTManager: creates a new JWT manager.
// Returns the JWT manager.
func NewJWTManager(secretKey []byte, expiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: secretKey,
		Expiry:    expiry,
	}
}

// GenerateToken: generates a new JWT token.
// Returns the JWT token and an error if the token generation fails.
func (m *JWTManager) GenerateToken(username string) (string, error) {
	if username == "" {
		return "", errors.New("username cannot be empty")
	}

	now := time.Now()
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken: validates a JWT token.
// Returns the username, token expiry time, and an error if the token validation fails.
func (m *JWTManager) ValidateToken(token string) (string, time.Time, error) {
	claims := &Claims{}

	parsedToken, err := jwt.ParseWithClaims(
		token,
		claims,
		func(token *jwt.Token) (interface{}, error) { return m.secretKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	if !parsedToken.Valid {
		return "", time.Time{}, errors.New("invalid token")
	}
	if claims.Username == "" {
		return "", time.Time{}, errors.New("missing username claim")
	}

	if claims.ExpiresAt == nil {
		return "", time.Time{}, errors.New("missing exp claim")
	}

	return claims.Username, claims.ExpiresAt.Time, nil
}
