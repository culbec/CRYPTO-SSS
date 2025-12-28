package test

import (
	"bytes"
	"testing"

	constants "github.com/culbec/CRYPTO-sss/src/backend/pkg"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/security"
)

func TestNewArgon2idHash(t *testing.T) {
	tests := []struct {
		name    string
		time    uint32
		memory  uint32
		threads uint8
		keyLen  uint32
		saltLen uint32
		wantNil bool
	}{
		{
			name:    "creates Argon2idHash with default parameters",
			time:    constants.ARGON2ID_DEFAULT_TIME,
			memory:  constants.ARGON2ID_DEFAULT_MEMORY,
			threads: constants.ARGON2ID_DEFAULT_THREADS,
			keyLen:  constants.ARGON2ID_DEFAULT_KEY_LEN,
			saltLen: constants.ARGON2ID_DEFAULT_SALT_LEN,
			wantNil: false,
		},
		{
			name:    "creates Argon2idHash with custom parameters",
			time:    3,
			memory:  4 * 1024,
			threads: 2,
			keyLen:  16,
			saltLen: 8,
			wantNil: false,
		},
		{
			name:    "creates Argon2idHash with zero values",
			time:    0,
			memory:  0,
			threads: 0,
			keyLen:  0,
			saltLen: 0,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := security.NewArgon2idHash(tt.time, tt.memory, tt.threads, tt.keyLen, tt.saltLen)
			if (got == nil) != tt.wantNil {
				t.Errorf("NewArgon2idHash() = %v, want nil = %v", got, tt.wantNil)
			}
			if got != nil {
				if got.Time != tt.time {
					t.Errorf("NewArgon2idHash() time = %v, want %v", got.Time, tt.time)
				}
				if got.Memory != tt.memory {
					t.Errorf("NewArgon2idHash() memory = %v, want %v", got.Memory, tt.memory)
				}
				if got.Threads != tt.threads {
					t.Errorf("NewArgon2idHash() threads = %v, want %v", got.Threads, tt.threads)
				}
				if got.KeyLen != tt.keyLen {
					t.Errorf("NewArgon2idHash() keyLen = %v, want %v", got.KeyLen, tt.keyLen)
				}
				if got.SaltLen != tt.saltLen {
					t.Errorf("NewArgon2idHash() saltLen = %v, want %v", got.SaltLen, tt.saltLen)
				}
			}
		})
	}
}

func TestArgon2idHash_GenerateHash(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)

	tests := []struct {
		name      string
		password  []byte
		salt      []byte
		wantError bool
	}{
		{
			name:      "generates hash with provided salt",
			password:  []byte("testpassword"),
			salt:      []byte("testsalt123456"),
			wantError: false,
		},
		{
			name:      "generates hash with auto-generated salt",
			password:  []byte("testpassword"),
			salt:      []byte{},
			wantError: false,
		},
		{
			name:      "generates hash with empty password",
			password:  []byte(""),
			salt:      []byte("testsalt123456"),
			wantError: false,
		},
		{
			name:      "generates hash with nil salt (treated as empty)",
			password:  []byte("testpassword"),
			salt:      nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashSalt, err := hasher.GenerateHash(tt.password, tt.salt)
			if (err != nil) != tt.wantError {
				t.Errorf("GenerateHash() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if hashSalt == nil {
					t.Errorf("GenerateHash() hashSalt = nil, want non-nil")
					return
				}
				if len(hashSalt.Hash) == 0 {
					t.Errorf("GenerateHash() hash = empty, want non-empty")
				}
				if len(hashSalt.Salt) == 0 {
					t.Errorf("GenerateHash() salt = empty, want non-empty")
				}
				// If salt was provided, it should match
				if len(tt.salt) > 0 && !bytes.Equal(hashSalt.Salt, tt.salt) {
					t.Errorf("GenerateHash() salt = %v, want %v", hashSalt.Salt, tt.salt)
				}
			}
		})
	}
}

func TestArgon2idHash_GenerateHash_Deterministic(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	password := []byte("testpassword")
	salt := []byte("testsalt123456")

	hashSalt1, err1 := hasher.GenerateHash(password, salt)
	if err1 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err1)
	}

	hashSalt2, err2 := hasher.GenerateHash(password, salt)
	if err2 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err2)
	}

	// Same password and salt should produce same hash
	if !bytes.Equal(hashSalt1.Hash, hashSalt2.Hash) {
		t.Errorf("GenerateHash() produced different hashes for same input: %v != %v", hashSalt1.Hash, hashSalt2.Hash)
	}

	if !bytes.Equal(hashSalt1.Salt, hashSalt2.Salt) {
		t.Errorf("GenerateHash() produced different salts: %v != %v", hashSalt1.Salt, hashSalt2.Salt)
	}
}

func TestArgon2idHash_ComparePasswords(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	password := []byte("testpassword")
	salt := []byte("testsalt123456")

	hashSalt, err := hasher.GenerateHash(password, salt)
	if err != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err)
	}

	tests := []struct {
		name      string
		password  []byte
		salt      []byte
		hash      []byte
		wantError bool
	}{
		{
			name:      "compares matching password correctly",
			password:  password,
			salt:      salt,
			hash:      hashSalt.Hash,
			wantError: false,
		},
		{
			name:      "rejects non-matching password",
			password:  []byte("wrongpassword"),
			salt:      salt,
			hash:      hashSalt.Hash,
			wantError: true,
		},
		{
			name:      "rejects password with wrong salt",
			password:  password,
			salt:      []byte("wrongsalt123456"),
			hash:      hashSalt.Hash,
			wantError: true,
		},
		{
			name:      "rejects empty password",
			password:  []byte(""),
			salt:      salt,
			hash:      hashSalt.Hash,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ComparePasswords(tt.password, tt.salt, tt.hash)
			if (err != nil) != tt.wantError {
				t.Errorf("ComparePasswords() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestArgon2idHash_ComparePasswords_WithAutoGeneratedSalt(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	password := []byte("testpassword")

	// Generate hash with auto-generated salt
	hashSalt, err := hasher.GenerateHash(password, []byte{})
	if err != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err)
	}

	// Should match with the same password and salt
	err = hasher.ComparePasswords(password, hashSalt.Salt, hashSalt.Hash)
	if err != nil {
		t.Errorf("ComparePasswords() error = %v, want nil for matching password", err)
	}

	// Should not match with wrong password
	err = hasher.ComparePasswords([]byte("wrongpassword"), hashSalt.Salt, hashSalt.Hash)
	if err == nil {
		t.Errorf("ComparePasswords() error = nil, want error for non-matching password")
	}
}

func TestArgon2idHash_GenerateHash_DifferentPasswords(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	salt := []byte("testsalt123456")

	hashSalt1, err1 := hasher.GenerateHash([]byte("password1"), salt)
	if err1 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err1)
	}

	hashSalt2, err2 := hasher.GenerateHash([]byte("password2"), salt)
	if err2 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err2)
	}

	// Different passwords should produce different hashes
	if bytes.Equal(hashSalt1.Hash, hashSalt2.Hash) {
		t.Errorf("GenerateHash() produced same hash for different passwords")
	}
}

func TestArgon2idHash_GenerateHash_DifferentSalts(t *testing.T) {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	password := []byte("testpassword")

	hashSalt1, err1 := hasher.GenerateHash(password, []byte("salt1"))
	if err1 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err1)
	}

	hashSalt2, err2 := hasher.GenerateHash(password, []byte("salt2"))
	if err2 != nil {
		t.Fatalf("GenerateHash() error = %v, want nil", err2)
	}

	// Same password with different salts should produce different hashes
	if bytes.Equal(hashSalt1.Hash, hashSalt2.Hash) {
		t.Errorf("GenerateHash() produced same hash for same password with different salts")
	}
}
