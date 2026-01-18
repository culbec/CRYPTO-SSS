// Package sss implements Shamir's (t,n) threshold secret sharing scheme
// with support for access structures suitable for voting applications.
//
// Shamir's Secret Sharing splits a secret into n shares such that any t shares
// can reconstruct the secret, but fewer than t shares reveal nothing about it.
//
// Access structures extend this to support complex authorization policies
// like "1 auditor AND 2 election officials" for vote reveal scenarios.
package sss

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"math/big"
)

var (
	// Prime is a 256-bit prime used for finite field arithmetic.
	// This is the prime 2^256 - 189, which is safe for cryptographic operations.
	Prime = new(big.Int).Sub(
		new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil),
		big.NewInt(189),
	)

	// ErrInsufficientShares is returned when not enough shares are provided for reconstruction.
	ErrInsufficientShares = errors.New("sss: insufficient shares for reconstruction")
	// ErrDuplicateShareIndex is returned when shares have duplicate indices.
	ErrDuplicateShareIndex = errors.New("sss: duplicate share index detected")
	// ErrInvalidThreshold is returned when threshold parameters are invalid.
	ErrInvalidThreshold = errors.New("sss: threshold must be > 0 and <= total shares")
	// ErrInvalidShareIndex is returned when a share index is invalid.
	ErrInvalidShareIndex = errors.New("sss: share index must be > 0")
	// ErrEmptySecret is returned when an empty secret is provided.
	ErrEmptySecret = errors.New("sss: secret cannot be empty")
)

// Share represents a single share of a split secret.
type Share struct {
	Index int      // X-coordinate (must be > 0)
	Value *big.Int // Y-coordinate (polynomial evaluation)
}

// ShareSet represents a collection of shares with metadata.
type ShareSet struct {
	Threshold int      // Minimum shares needed for reconstruction
	Total     int      // Total number of shares created
	Shares    []*Share // The actual shares
}

// polynomial represents a polynomial over the finite field GF(Prime).
type polynomial struct {
	coefficients []*big.Int // coefficients[0] is the constant term (the secret)
}

// newPolynomial creates a random polynomial of given degree with specified constant term.
func newPolynomial(degree int, constant *big.Int) (*polynomial, error) {
	coefficients := make([]*big.Int, degree+1)
	coefficients[0] = new(big.Int).Set(constant)

	for i := 1; i <= degree; i++ {
		coef, err := randomFieldElement()
		if err != nil {
			return nil, err
		}
		coefficients[i] = coef
	}

	return &polynomial{coefficients: coefficients}, nil
}

// evaluate computes the polynomial at point x using Horner's method.
func (p *polynomial) evaluate(x *big.Int) *big.Int {
	result := new(big.Int).Set(p.coefficients[len(p.coefficients)-1])

	for i := len(p.coefficients) - 2; i >= 0; i-- {
		result.Mul(result, x)
		result.Add(result, p.coefficients[i])
		result.Mod(result, Prime)
	}

	return result
}

// randomFieldElement generates a random element in GF(Prime).
func randomFieldElement() (*big.Int, error) {
	max := new(big.Int).Sub(Prime, big.NewInt(1))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Split splits a secret into n shares with threshold t.
// Any t shares can reconstruct the secret, but t-1 shares reveal nothing.
//
// Parameters:
//   - secret: the secret bytes to split
//   - threshold: minimum number of shares needed (t)
//   - total: total number of shares to create (n)
//
// Returns a ShareSet containing all shares and metadata.
func Split(secret []byte, threshold, total int) (*ShareSet, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}
	if threshold <= 0 || threshold > total {
		return nil, ErrInvalidThreshold
	}

	// Convert secret to big.Int
	secretInt := new(big.Int).SetBytes(secret)

	// Ensure secret is within field
	secretInt.Mod(secretInt, Prime)

	// Create random polynomial of degree (threshold - 1)
	poly, err := newPolynomial(threshold-1, secretInt)
	if err != nil {
		return nil, err
	}

	// Generate shares by evaluating polynomial at points 1, 2, ..., total
	shares := make([]*Share, total)
	for i := 0; i < total; i++ {
		x := big.NewInt(int64(i + 1))
		y := poly.evaluate(x)
		shares[i] = &Share{
			Index: i + 1,
			Value: y,
		}
	}

	return &ShareSet{
		Threshold: threshold,
		Total:     total,
		Shares:    shares,
	}, nil
}

// Combine reconstructs the secret from shares using Lagrange interpolation.
// At least threshold shares are required for successful reconstruction.
//
// Parameters:
//   - shares: slice of shares to use for reconstruction
//   - threshold: the minimum number of shares required
//
// Returns the reconstructed secret bytes.
func Combine(shares []*Share, threshold int) ([]byte, error) {
	if len(shares) < threshold {
		return nil, ErrInsufficientShares
	}

	// Check for duplicate indices
	seen := make(map[int]bool)
	for _, share := range shares {
		if share.Index <= 0 {
			return nil, ErrInvalidShareIndex
		}
		if seen[share.Index] {
			return nil, ErrDuplicateShareIndex
		}
		seen[share.Index] = true
	}

	// Use only the first 'threshold' shares
	usedShares := shares[:threshold]

	// Lagrange interpolation to find f(0) = secret
	secret := new(big.Int)

	for i, share := range usedShares {
		xi := big.NewInt(int64(share.Index))
		yi := share.Value

		// Compute Lagrange basis polynomial L_i(0)
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j, other := range usedShares {
			if i == j {
				continue
			}
			xj := big.NewInt(int64(other.Index))

			// numerator *= (0 - xj) = -xj
			numerator.Mul(numerator, new(big.Int).Neg(xj))
			numerator.Mod(numerator, Prime)

			// denominator *= (xi - xj)
			diff := new(big.Int).Sub(xi, xj)
			denominator.Mul(denominator, diff)
			denominator.Mod(denominator, Prime)
		}

		// Compute L_i(0) = numerator / denominator (mod Prime)
		// Division in modular arithmetic is multiplication by inverse
		denomInv := new(big.Int).ModInverse(denominator, Prime)
		lagrangeBasis := new(big.Int).Mul(numerator, denomInv)
		lagrangeBasis.Mod(lagrangeBasis, Prime)

		// Add yi * L_i(0) to the result
		term := new(big.Int).Mul(yi, lagrangeBasis)
		term.Mod(term, Prime)
		secret.Add(secret, term)
		secret.Mod(secret, Prime)
	}

	return secret.Bytes(), nil
}

// ComputeShareCommitment computes a SHA-256 commitment for a share.
// This can be used to verify share integrity without revealing the share value.
func ComputeShareCommitment(share *Share) []byte {
	h := sha256.New()
	h.Write([]byte{byte(share.Index >> 24), byte(share.Index >> 16), byte(share.Index >> 8), byte(share.Index)})
	h.Write(share.Value.Bytes())
	return h.Sum(nil)
}

// VerifyShareCommitment verifies that a share matches its commitment.
// Uses constant-time comparison to prevent timing attacks.
func VerifyShareCommitment(share *Share, commitment []byte) bool {
	computed := ComputeShareCommitment(share)
	return subtle.ConstantTimeCompare(computed, commitment) == 1
}
