package test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	security_jwt "github.com/culbec/CRYPTO-sss/src/backend/pkg/security/jwt"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name      string
		secretKey []byte
		expiry    time.Duration
		wantNil   bool
	}{
		{
			name:      "creates JWT manager with valid parameters",
			secretKey: []byte("test-secret-key"),
			expiry:    time.Hour,
			wantNil:   false,
		},
		{
			name:      "creates JWT manager with empty secret key",
			secretKey: []byte(""),
			expiry:    time.Hour,
			wantNil:   false,
		},
		{
			name:      "creates JWT manager with zero expiry",
			secretKey: []byte("test-secret-key"),
			expiry:    0,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security_jwt.NewJWTManager(tt.secretKey, tt.expiry)
			if (got == nil) != tt.wantNil {
				t.Errorf("NewJWTManager() = %v, want nil = %v", got, tt.wantNil)
			}
			if got != nil {
				if got.Expiry != tt.expiry {
					t.Errorf("NewJWTManager() expiry = %v, want %v", got.Expiry, tt.expiry)
				}
			}
		})
	}
}

func TestJWTManager_GenerateToken(t *testing.T) {
	secretKey := []byte("test-secret-key")
	expiry := time.Hour
	manager := security_jwt.NewJWTManager(secretKey, expiry)

	tests := []struct {
		name      string
		username  string
		wantError bool
	}{
		{
			name:      "generates token for valid username",
			username:  "testuser",
			wantError: false,
		},
		{
			name:      "generates token for empty username",
			username:  "",
			wantError: false,
		},
		{
			name:      "generates token for username with special characters",
			username:  "user@example.com",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.username)
			if (err != nil) != tt.wantError {
				t.Errorf("GenerateToken() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if token == "" {
					t.Errorf("GenerateToken() token = empty string, want non-empty")
				}
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	secretKey := []byte("test-secret-key")
	expiry := time.Hour
	manager := security_jwt.NewJWTManager(secretKey, expiry)

	tests := []struct {
		name      string
		setup     func() string
		wantUser  string
		wantError bool
	}{
		{
			name: "validates token with correct secret key",
			setup: func() string {
				token, _ := manager.GenerateToken("testuser")
				return token
			},
			wantUser:  "testuser",
			wantError: false,
		},
		{
			name: "validates token with empty username",
			setup: func() string {
				token, _ := manager.GenerateToken("")
				return token
			},
			wantUser:  "",
			wantError: false,
		},
		{
			name: "rejects invalid token string",
			setup: func() string {
				return "invalid.token.string"
			},
			wantUser:  "",
			wantError: true,
		},
		{
			name: "rejects token with wrong secret key",
			setup: func() string {
				wrongManager := security_jwt.NewJWTManager([]byte("wrong-secret-key"), expiry)
				token, _ := wrongManager.GenerateToken("testuser")
				return token
			},
			wantUser:  "",
			wantError: true,
		},
		{
			name: "rejects empty token string",
			setup: func() string {
				return ""
			},
			wantUser:  "",
			wantError: true,
		},
		{
			name: "rejects token with wrong signing method",
			setup: func() string {
				// Create a token with RS256 instead of HS256
				claims := jwt.MapClaims{
					"username": "testuser",
					"exp":      time.Now().Add(expiry).Unix(),
				}
				_ = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				// This will fail to sign, but we can test the validation
				return "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InRlc3R1c2VyIn0.invalid"
			},
			wantUser:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setup()
			username, err := manager.ValidateToken(token)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateToken() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && username != tt.wantUser {
				t.Errorf("ValidateToken() username = %v, want %v", username, tt.wantUser)
			}
		})
	}
}

func TestJWTManager_GenerateAndValidateToken(t *testing.T) {
	secretKey := []byte("test-secret-key")
	expiry := time.Hour
	manager := security_jwt.NewJWTManager(secretKey, expiry)

	tests := []struct {
		name     string
		username string
	}{
		{
			name:     "generates and validates token for regular username",
			username: "testuser",
		},
		{
			name:     "generates and validates token for username with special chars",
			username: "user@example.com",
		},
		{
			name:     "generates and validates token for empty username",
			username: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.username)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v, want nil", err)
			}

			username, err := manager.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v, want nil", err)
			}

			if username != tt.username {
				t.Errorf("ValidateToken() username = %v, want %v", username, tt.username)
			}
		})
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	secretKey := []byte("test-secret-key")
	expiry := -time.Hour // Negative expiry means token is already expired
	manager := security_jwt.NewJWTManager(secretKey, expiry)

	token, err := manager.GenerateToken("testuser")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v, want nil", err)
	}

	// Wait a bit to ensure token is expired
	time.Sleep(100 * time.Millisecond)

	username, err := manager.ValidateToken(token)
	if err == nil {
		t.Errorf("ValidateToken() error = nil, want error for expired token")
	}
	if username != "" {
		t.Errorf("ValidateToken() username = %v, want empty string for expired token", username)
	}
}
