package test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/pkg/sss"
)

// TestSplit tests basic secret splitting functionality.
func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		secret    []byte
		threshold int
		total     int
		wantErr   error
	}{
		{
			name:      "valid 2-of-3 split",
			secret:    []byte("my secret key for voting"),
			threshold: 2,
			total:     3,
			wantErr:   nil,
		},
		{
			name:      "valid 3-of-5 split",
			secret:    []byte("another secret"),
			threshold: 3,
			total:     5,
			wantErr:   nil,
		},
		{
			name:      "valid 1-of-1 split",
			secret:    []byte("minimal secret"),
			threshold: 1,
			total:     1,
			wantErr:   nil,
		},
		{
			name:      "valid n-of-n split",
			secret:    []byte("all shares needed"),
			threshold: 5,
			total:     5,
			wantErr:   nil,
		},
		{
			name:      "empty secret should fail",
			secret:    []byte{},
			threshold: 2,
			total:     3,
			wantErr:   sss.ErrEmptySecret,
		},
		{
			name:      "nil secret should fail",
			secret:    nil,
			threshold: 2,
			total:     3,
			wantErr:   sss.ErrEmptySecret,
		},
		{
			name:      "threshold > total should fail",
			secret:    []byte("secret"),
			threshold: 5,
			total:     3,
			wantErr:   sss.ErrInvalidThreshold,
		},
		{
			name:      "zero threshold should fail",
			secret:    []byte("secret"),
			threshold: 0,
			total:     3,
			wantErr:   sss.ErrInvalidThreshold,
		},
		{
			name:      "negative threshold should fail",
			secret:    []byte("secret"),
			threshold: -1,
			total:     3,
			wantErr:   sss.ErrInvalidThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shareSet, err := sss.Split(tt.secret, tt.threshold, tt.total)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Split() expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("Split() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Split() unexpected error: %v", err)
				return
			}

			if shareSet == nil {
				t.Error("Split() returned nil ShareSet")
				return
			}

			if shareSet.Threshold != tt.threshold {
				t.Errorf("Split() threshold = %d, want %d", shareSet.Threshold, tt.threshold)
			}
			if shareSet.Total != tt.total {
				t.Errorf("Split() total = %d, want %d", shareSet.Total, tt.total)
			}
			if len(shareSet.Shares) != tt.total {
				t.Errorf("Split() shares count = %d, want %d", len(shareSet.Shares), tt.total)
			}

			// Verify all shares have unique indices
			indices := make(map[int]bool)
			for _, share := range shareSet.Shares {
				if share.Index <= 0 {
					t.Errorf("Split() share index = %d, want > 0", share.Index)
				}
				if indices[share.Index] {
					t.Errorf("Split() duplicate share index: %d", share.Index)
				}
				indices[share.Index] = true
			}
		})
	}
}

// TestCombine tests secret reconstruction from shares.
func TestCombine(t *testing.T) {
	secret := []byte("secret ballot encryption key")

	// Create a 3-of-5 share set
	shareSet, err := sss.Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	tests := []struct {
		name       string
		shares     []*sss.Share
		threshold  int
		wantErr    error
		wantSecret bool
	}{
		{
			name:       "exact threshold shares reconstruct secret",
			shares:     shareSet.Shares[:3],
			threshold:  3,
			wantErr:    nil,
			wantSecret: true,
		},
		{
			name:       "more than threshold shares reconstruct secret",
			shares:     shareSet.Shares[:4],
			threshold:  3,
			wantErr:    nil,
			wantSecret: true,
		},
		{
			name:       "all shares reconstruct secret",
			shares:     shareSet.Shares,
			threshold:  3,
			wantErr:    nil,
			wantSecret: true,
		},
		{
			name:       "fewer than threshold shares fails",
			shares:     shareSet.Shares[:2],
			threshold:  3,
			wantErr:    sss.ErrInsufficientShares,
			wantSecret: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconstructed, err := sss.Combine(tt.shares, tt.threshold)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Combine() expected error %v, got nil", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("Combine() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Combine() unexpected error: %v", err)
				return
			}

			if tt.wantSecret {
				// Secret may have leading zeros stripped, so compare big.Int
				originalInt := new(big.Int).SetBytes(secret)
				originalInt.Mod(originalInt, sss.Prime)
				reconstructedInt := new(big.Int).SetBytes(reconstructed)

				if originalInt.Cmp(reconstructedInt) != 0 {
					t.Errorf("Combine() reconstructed secret does not match original")
				}
			}
		})
	}
}

// TestCombineWithDifferentShareSubsets tests that any t shares work.
func TestCombineWithDifferentShareSubsets(t *testing.T) {
	secret := []byte("test secret for subset verification")
	threshold := 3
	total := 5

	shareSet, err := sss.Split(secret, threshold, total)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	// Test different combinations of 3 shares from 5
	combinations := [][]int{
		{0, 1, 2},
		{0, 1, 3},
		{0, 1, 4},
		{0, 2, 3},
		{0, 2, 4},
		{0, 3, 4},
		{1, 2, 3},
		{1, 2, 4},
		{1, 3, 4},
		{2, 3, 4},
	}

	originalInt := new(big.Int).SetBytes(secret)
	originalInt.Mod(originalInt, sss.Prime)

	for _, combo := range combinations {
		shares := []*sss.Share{
			shareSet.Shares[combo[0]],
			shareSet.Shares[combo[1]],
			shareSet.Shares[combo[2]],
		}

		reconstructed, err := sss.Combine(shares, threshold)
		if err != nil {
			t.Errorf("Combine() with shares %v failed: %v", combo, err)
			continue
		}

		reconstructedInt := new(big.Int).SetBytes(reconstructed)
		if originalInt.Cmp(reconstructedInt) != 0 {
			t.Errorf("Combine() with shares %v produced wrong secret", combo)
		}
	}
}

// TestCombineDuplicateShares tests that duplicate shares are rejected.
func TestCombineDuplicateShares(t *testing.T) {
	secret := []byte("duplicate test secret")
	shareSet, err := sss.Split(secret, 2, 3)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	// Create shares with duplicate indices
	duplicateShares := []*sss.Share{
		shareSet.Shares[0],
		shareSet.Shares[0], // Duplicate
	}

	_, err = sss.Combine(duplicateShares, 2)
	if err != sss.ErrDuplicateShareIndex {
		t.Errorf("Combine() with duplicate shares: error = %v, want %v", err, sss.ErrDuplicateShareIndex)
	}
}

// TestCombineInvalidShareIndex tests that invalid share indices are rejected.
func TestCombineInvalidShareIndex(t *testing.T) {
	invalidShares := []*sss.Share{
		{Index: 0, Value: big.NewInt(100)},  // Invalid: index = 0
		{Index: 1, Value: big.NewInt(200)},
	}

	_, err := sss.Combine(invalidShares, 2)
	if err != sss.ErrInvalidShareIndex {
		t.Errorf("Combine() with invalid index: error = %v, want %v", err, sss.ErrInvalidShareIndex)
	}
}

// TestShareCommitment tests commitment computation and verification.
func TestShareCommitment(t *testing.T) {
	shareSet, err := sss.Split([]byte("commitment test"), 2, 3)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	for _, share := range shareSet.Shares {
		commitment := sss.ComputeShareCommitment(share)

		if len(commitment) != 32 { // SHA-256 output
			t.Errorf("ComputeShareCommitment() length = %d, want 32", len(commitment))
		}

		if !sss.VerifyShareCommitment(share, commitment) {
			t.Error("VerifyShareCommitment() returned false for valid commitment")
		}

		// Tamper with commitment and verify it fails
		tamperedCommitment := make([]byte, len(commitment))
		copy(tamperedCommitment, commitment)
		tamperedCommitment[0] ^= 0xFF

		if sss.VerifyShareCommitment(share, tamperedCommitment) {
			t.Error("VerifyShareCommitment() returned true for tampered commitment")
		}
	}
}

// TestShareCommitmentDeterministic tests that commitments are deterministic.
func TestShareCommitmentDeterministic(t *testing.T) {
	share := &sss.Share{
		Index: 42,
		Value: big.NewInt(123456789),
	}

	commitment1 := sss.ComputeShareCommitment(share)
	commitment2 := sss.ComputeShareCommitment(share)

	if !bytes.Equal(commitment1, commitment2) {
		t.Error("ComputeShareCommitment() produced different results for same input")
	}
}

// TestLargeSecret tests splitting and combining large secrets.
func TestLargeSecret(t *testing.T) {
	// Create a 32-byte secret (typical AES key size)
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	shareSet, err := sss.Split(secret, 3, 5)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	reconstructed, err := sss.Combine(shareSet.Shares[:3], 3)
	if err != nil {
		t.Fatalf("Combine() failed: %v", err)
	}

	originalInt := new(big.Int).SetBytes(secret)
	originalInt.Mod(originalInt, sss.Prime)
	reconstructedInt := new(big.Int).SetBytes(reconstructed)

	if originalInt.Cmp(reconstructedInt) != 0 {
		t.Error("Large secret reconstruction failed")
	}
}

// TestSplitProducesUniqueShares tests that repeated splits produce different shares.
func TestSplitProducesUniqueShares(t *testing.T) {
	secret := []byte("random test secret")

	shareSet1, err := sss.Split(secret, 2, 3)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	shareSet2, err := sss.Split(secret, 2, 3)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	// Shares should be different due to random polynomial coefficients
	allSame := true
	for i := range shareSet1.Shares {
		if shareSet1.Shares[i].Value.Cmp(shareSet2.Shares[i].Value) != 0 {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Split() produced identical shares for separate calls (should be random)")
	}
}

// TestPrimeFieldBoundary tests secrets near the prime boundary.
func TestPrimeFieldBoundary(t *testing.T) {
	// Test with a secret that's close to the prime modulus
	largeSecret := make([]byte, 32)
	for i := range largeSecret {
		largeSecret[i] = 0xFF
	}

	shareSet, err := sss.Split(largeSecret, 2, 3)
	if err != nil {
		t.Fatalf("Split() failed: %v", err)
	}

	reconstructed, err := sss.Combine(shareSet.Shares[:2], 2)
	if err != nil {
		t.Fatalf("Combine() failed: %v", err)
	}

	// The secret should be reduced mod Prime
	originalInt := new(big.Int).SetBytes(largeSecret)
	originalInt.Mod(originalInt, sss.Prime)
	reconstructedInt := new(big.Int).SetBytes(reconstructed)

	if originalInt.Cmp(reconstructedInt) != 0 {
		t.Error("Boundary secret reconstruction failed")
	}
}

// TestVerifyShareCommitmentWrongLength tests rejection of wrong-length commitments.
func TestVerifyShareCommitmentWrongLength(t *testing.T) {
	share := &sss.Share{
		Index: 1,
		Value: big.NewInt(12345),
	}

	tests := []struct {
		name       string
		commitment []byte
		expected   bool
	}{
		{"empty commitment", []byte{}, false},
		{"too short (16 bytes)", make([]byte, 16), false},
		{"too short (31 bytes)", make([]byte, 31), false},
		{"correct length (32 bytes)", sss.ComputeShareCommitment(share), true},
		{"too long (33 bytes)", make([]byte, 33), false},
		{"too long (64 bytes)", make([]byte, 64), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sss.VerifyShareCommitment(share, tt.commitment)
			if result != tt.expected {
				t.Errorf("VerifyShareCommitment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestVerifyShareCommitmentTimingSafe tests that verification is timing-safe.
// This is a basic sanity check - true timing analysis requires specialized tools.
func TestVerifyShareCommitmentTimingSafe(t *testing.T) {
	share := &sss.Share{
		Index: 1,
		Value: big.NewInt(999999999),
	}
	validCommitment := sss.ComputeShareCommitment(share)

	// Create commitments with different first bytes that differ
	invalidCommitments := make([][]byte, 256)
	for i := 0; i < 256; i++ {
		invalid := make([]byte, len(validCommitment))
		copy(invalid, validCommitment)
		invalid[0] = byte(i)
		invalidCommitments[i] = invalid
	}

	// All invalid commitments should be rejected
	for i, invalid := range invalidCommitments {
		if bytes.Equal(invalid, validCommitment) {
			continue // Skip the valid one
		}
		if sss.VerifyShareCommitment(share, invalid) {
			t.Errorf("VerifyShareCommitment() accepted invalid commitment with first byte %d", i)
		}
	}

	// Valid commitment should be accepted
	if !sss.VerifyShareCommitment(share, validCommitment) {
		t.Error("VerifyShareCommitment() rejected valid commitment")
	}
}

// TestShareCommitmentDifferentShares tests that different shares produce different commitments.
func TestShareCommitmentDifferentShares(t *testing.T) {
	share1 := &sss.Share{Index: 1, Value: big.NewInt(100)}
	share2 := &sss.Share{Index: 2, Value: big.NewInt(100)}
	share3 := &sss.Share{Index: 1, Value: big.NewInt(200)}

	commitment1 := sss.ComputeShareCommitment(share1)
	commitment2 := sss.ComputeShareCommitment(share2)
	commitment3 := sss.ComputeShareCommitment(share3)

	// Different indices should produce different commitments
	if bytes.Equal(commitment1, commitment2) {
		t.Error("Different share indices should produce different commitments")
	}

	// Different values should produce different commitments
	if bytes.Equal(commitment1, commitment3) {
		t.Error("Different share values should produce different commitments")
	}

	// All three should be unique
	if bytes.Equal(commitment2, commitment3) {
		t.Error("All commitments should be unique")
	}
}
