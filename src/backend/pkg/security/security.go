package security

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/argon2"
)


// Argon2idHash: struct to hold the Argon2id hash configuration.
type Argon2idHash struct {
	Time    uint32 // number of passes over the memory
	Memory  uint32 // memory size in KiB
	Threads uint8  // number of threads
	KeyLen  uint32 // key length
	SaltLen uint32 // salt length
}

// HashSalt: struct to hold the hash and salt.
type HashSalt struct {
	Hash []byte // hashed password
	Salt []byte // salt used for hashing
}

// NewArgon2idHash: creates a new Argon2id hash.
// Returns the Argon2id hash.
func NewArgon2idHash(time, memory uint32, threads uint8, keyLen, saltLen uint32) *Argon2idHash {
	return &Argon2idHash{
		Time:    time,
		Memory:  memory,
		Threads: threads,
		KeyLen:  keyLen,
		SaltLen: saltLen,
	}
}

// secret: generates a random secret.
// Returns the secret and an error if the secret generation fails.
func secret(len uint32) ([]byte, error) {
	secretBytes := make([]byte, len)

	_, err := rand.Read(secretBytes)
	if err != nil {
		return nil, err
	}

	return secretBytes, nil
}

// GenerateHash: generates a hash for a given password and salt.
// Returns the hash and an error if the hash generation fails.
func (a *Argon2idHash) GenerateHash(password, salt []byte) (*HashSalt, error) {
	var err error

	if len(salt) == 0 {
		salt, err = secret(a.SaltLen)
		if err != nil {
			return nil, err
		}
	}

	hash := hex.EncodeToString(argon2.IDKey(password, salt, a.Time, a.Memory, a.Threads, a.KeyLen))
	return &HashSalt{Hash: []byte(hash), Salt: salt}, nil
}

// ComparePasswords: compares a password and a hash.
// Returns an error if the passwords do not match.
func (a *Argon2idHash) ComparePasswords(password, salt, hash []byte) error {
	hashSalt, err := a.GenerateHash(password, salt)
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, hashSalt.Hash) {
		return errors.New("authentication failed: password verification failed")
	}

	return nil
}