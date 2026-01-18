package test

import (
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/helpers"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
)

func TestValidateUserRole(t *testing.T) {
	tests := []struct {
		name     string
		role     types.UserRole
		expected bool
	}{
		{
			name:     "valid voter role",
			role:     types.RoleVoter,
			expected: true,
		},
		{
			name:     "valid auditor role",
			role:     types.RoleAuditor,
			expected: true,
		},
		{
			name:     "valid official role",
			role:     types.RoleOfficial,
			expected: true,
		},
		{
			name:     "valid admin role",
			role:     types.RoleAdmin,
			expected: true,
		},
		{
			name:     "invalid empty role",
			role:     types.UserRole(""),
			expected: false,
		},
		{
			name:     "invalid arbitrary role",
			role:     types.UserRole("superuser"),
			expected: false,
		},
		{
			name:     "invalid role with typo",
			role:     types.UserRole("votor"),
			expected: false,
		},
		{
			name:     "invalid role with case mismatch",
			role:     types.UserRole("VOTER"),
			expected: false,
		},
		{
			name:     "invalid role with spaces",
			role:     types.UserRole(" voter "),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ValidateUserRole(tt.role)
			if result != tt.expected {
				t.Errorf("ValidateUserRole(%q) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestUserRoleConstants(t *testing.T) {
	// Verify role constants have expected values
	tests := []struct {
		role     types.UserRole
		expected string
	}{
		{types.RoleVoter, "voter"},
		{types.RoleAuditor, "auditor"},
		{types.RoleOfficial, "official"},
		{types.RoleAdmin, "admin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("Role constant %v = %q, want %q", tt.role, string(tt.role), tt.expected)
			}
		})
	}
}

func TestHelperErrors(t *testing.T) {
	// Verify error constants are defined
	if helpers.ErrUserNotFound == nil {
		t.Error("ErrUserNotFound should not be nil")
	}
	if helpers.ErrPollNotFound == nil {
		t.Error("ErrPollNotFound should not be nil")
	}

	// Verify error messages
	if helpers.ErrUserNotFound.Error() != "user not found" {
		t.Errorf("ErrUserNotFound message = %q, want %q", helpers.ErrUserNotFound.Error(), "user not found")
	}
	if helpers.ErrPollNotFound.Error() != "poll not found" {
		t.Errorf("ErrPollNotFound message = %q, want %q", helpers.ErrPollNotFound.Error(), "poll not found")
	}
}
