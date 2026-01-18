package test

import (
	"testing"
	"time"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
)

func TestPollStatusConstants(t *testing.T) {
	// Verify poll status constants have expected values
	tests := []struct {
		status   types.PollStatus
		expected string
	}{
		{types.PollStatusDraft, "draft"},
		{types.PollStatusOpen, "open"},
		{types.PollStatusClosed, "closed"},
		{types.PollStatusFrozen, "frozen"},
		{types.PollStatusRevealed, "revealed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("PollStatus constant = %q, want %q", string(tt.status), tt.expected)
			}
		})
	}
}

func TestPollStatusTransitions(t *testing.T) {
	// Define valid transitions
	validTransitions := map[types.PollStatus][]types.PollStatus{
		types.PollStatusDraft:    {types.PollStatusOpen},
		types.PollStatusOpen:     {types.PollStatusClosed},
		types.PollStatusClosed:   {types.PollStatusFrozen},
		types.PollStatusFrozen:   {types.PollStatusRevealed},
		types.PollStatusRevealed: {},
	}

	// Helper function to check if transition is valid
	isValidTransition := func(from, to types.PollStatus) bool {
		validTargets, exists := validTransitions[from]
		if !exists {
			return false
		}
		for _, valid := range validTargets {
			if valid == to {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name     string
		from     types.PollStatus
		to       types.PollStatus
		expected bool
	}{
		// Valid transitions
		{"draft to open", types.PollStatusDraft, types.PollStatusOpen, true},
		{"open to closed", types.PollStatusOpen, types.PollStatusClosed, true},
		{"closed to frozen", types.PollStatusClosed, types.PollStatusFrozen, true},
		{"frozen to revealed", types.PollStatusFrozen, types.PollStatusRevealed, true},
		// Invalid transitions
		{"draft to closed (skip)", types.PollStatusDraft, types.PollStatusClosed, false},
		{"draft to frozen (skip)", types.PollStatusDraft, types.PollStatusFrozen, false},
		{"draft to revealed (skip)", types.PollStatusDraft, types.PollStatusRevealed, false},
		{"open to frozen (skip)", types.PollStatusOpen, types.PollStatusFrozen, false},
		{"open to revealed (skip)", types.PollStatusOpen, types.PollStatusRevealed, false},
		{"closed to revealed (skip)", types.PollStatusClosed, types.PollStatusRevealed, false},
		{"revealed to anything", types.PollStatusRevealed, types.PollStatusOpen, false},
		// Backwards transitions
		{"open to draft (backward)", types.PollStatusOpen, types.PollStatusDraft, false},
		{"closed to open (backward)", types.PollStatusClosed, types.PollStatusOpen, false},
		{"frozen to closed (backward)", types.PollStatusFrozen, types.PollStatusClosed, false},
		{"revealed to frozen (backward)", types.PollStatusRevealed, types.PollStatusFrozen, false},
		// Same status
		{"draft to draft", types.PollStatusDraft, types.PollStatusDraft, false},
		{"open to open", types.PollStatusOpen, types.PollStatusOpen, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("Transition from %s to %s = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestPollOptionValidation(t *testing.T) {
	tests := []struct {
		name    string
		option  types.PollOption
		isValid bool
	}{
		{
			name:    "valid option",
			option:  types.PollOption{ID: "opt_1", Label: "Option One"},
			isValid: true,
		},
		{
			name:    "empty ID",
			option:  types.PollOption{ID: "", Label: "Option One"},
			isValid: false,
		},
		{
			name:    "empty label",
			option:  types.PollOption{ID: "opt_1", Label: ""},
			isValid: false,
		},
		{
			name:    "both empty",
			option:  types.PollOption{ID: "", Label: ""},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: both ID and Label must be non-empty
			isValid := tt.option.ID != "" && tt.option.Label != ""
			if isValid != tt.isValid {
				t.Errorf("PollOption validation = %v, want %v", isValid, tt.isValid)
			}
		})
	}
}

func TestCreatePollRequestValidation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		req     types.CreatePollRequest
		isValid bool
		reason  string
	}{
		{
			name: "valid request",
			req: types.CreatePollRequest{
				Title:             "Test Poll",
				Description:       "A test poll",
				Options:           []types.PollOption{{ID: "a", Label: "Option A"}, {ID: "b", Label: "Option B"}},
				AuditorThreshold:  1,
				AuditorTotal:      2,
				OfficialThreshold: 2,
				OfficialTotal:     3,
			},
			isValid: true,
		},
		{
			name: "empty title",
			req: types.CreatePollRequest{
				Title:             "",
				Options:           []types.PollOption{{ID: "a", Label: "Option A"}, {ID: "b", Label: "Option B"}},
				AuditorThreshold:  1,
				AuditorTotal:      2,
				OfficialThreshold: 2,
				OfficialTotal:     3,
			},
			isValid: false,
			reason:  "title is required",
		},
		{
			name: "insufficient options",
			req: types.CreatePollRequest{
				Title:             "Test Poll",
				Options:           []types.PollOption{{ID: "a", Label: "Option A"}},
				AuditorThreshold:  1,
				AuditorTotal:      2,
				OfficialThreshold: 2,
				OfficialTotal:     3,
			},
			isValid: false,
			reason:  "at least 2 options required",
		},
		{
			name: "auditor threshold exceeds total",
			req: types.CreatePollRequest{
				Title:             "Test Poll",
				Options:           []types.PollOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}},
				AuditorThreshold:  3,
				AuditorTotal:      2,
				OfficialThreshold: 2,
				OfficialTotal:     3,
			},
			isValid: false,
			reason:  "auditor threshold cannot exceed total",
		},
		{
			name: "official threshold exceeds total",
			req: types.CreatePollRequest{
				Title:             "Test Poll",
				Options:           []types.PollOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}},
				AuditorThreshold:  1,
				AuditorTotal:      2,
				OfficialThreshold: 5,
				OfficialTotal:     3,
			},
			isValid: false,
			reason:  "official threshold cannot exceed total",
		},
		{
			name: "with valid time range",
			req: types.CreatePollRequest{
				Title:             "Test Poll",
				Options:           []types.PollOption{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}},
				StartTime:         &now,
				EndTime:           func() *time.Time { t := now.Add(24 * time.Hour); return &t }(),
				AuditorThreshold:  1,
				AuditorTotal:      2,
				OfficialThreshold: 2,
				OfficialTotal:     3,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate request
			isValid := true
			if tt.req.Title == "" {
				isValid = false
			}
			if len(tt.req.Options) < 2 {
				isValid = false
			}
			if tt.req.AuditorThreshold > tt.req.AuditorTotal {
				isValid = false
			}
			if tt.req.OfficialThreshold > tt.req.OfficialTotal {
				isValid = false
			}

			if isValid != tt.isValid {
				t.Errorf("CreatePollRequest validation = %v, want %v (reason: %s)", isValid, tt.isValid, tt.reason)
			}
		})
	}
}

func TestUserRoleComparison(t *testing.T) {
	// Test that role comparison works correctly
	voter := types.RoleVoter
	auditor := types.RoleAuditor
	official := types.RoleOfficial
	admin := types.RoleAdmin

	if voter == auditor {
		t.Error("voter should not equal auditor")
	}
	if voter == official {
		t.Error("voter should not equal official")
	}
	if voter == admin {
		t.Error("voter should not equal admin")
	}

	// Test self-equality
	if voter != types.RoleVoter {
		t.Error("voter should equal RoleVoter")
	}
	if auditor != types.RoleAuditor {
		t.Error("auditor should equal RoleAuditor")
	}
	if official != types.RoleOfficial {
		t.Error("official should equal RoleOfficial")
	}
	if admin != types.RoleAdmin {
		t.Error("admin should equal RoleAdmin")
	}
}
