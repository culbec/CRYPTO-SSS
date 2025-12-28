package security_jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(m.Expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken: validates a JWT token.
// Returns the username and an error if the token validation fails.
func (m *JWTManager) ValidateToken(token string) (string, error) {
	// Parse the token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrInvalidKey
		}

		return m.secretKey, nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)

	if !ok || !parsedToken.Valid {
		return "", jwt.ErrInvalidKey
	}

	usernameClaim, ok := claims["username"]
	if !ok {
		return "", jwt.ErrInvalidKey
	}

	username, ok := usernameClaim.(string)
	if !ok {
		return "", jwt.ErrInvalidType
	}

	return username, nil
}
