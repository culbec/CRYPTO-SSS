package test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/pkg/sss"
)

// TestAccessStructure_AddGroup tests group addition validation.
func TestAccessStructure_AddGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		threshold int
		total     int
		wantErr   bool
	}{
		{
			name:      "valid group 2-of-3",
			groupName: "officials",
			threshold: 2,
			total:     3,
			wantErr:   false,
		},
		{
			name:      "valid group 1-of-1",
			groupName: "auditor",
			threshold: 1,
			total:     1,
			wantErr:   false,
		},
		{
			name:      "invalid threshold > total",
			groupName: "invalid",
			threshold: 5,
			total:     3,
			wantErr:   true,
		},
		{
			name:      "invalid zero threshold",
			groupName: "zero",
			threshold: 0,
			total:     3,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := sss.NewAccessStructure([]byte("test secret"))
			err := as.AddGroup(tt.groupName, tt.threshold, tt.total)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAccessStructure_VotingScenario tests the standard voting scenario.
func TestAccessStructure_VotingScenario(t *testing.T) {
	secret := []byte("election master key 2024")

	// Create voting scenario: 1 auditor AND 2 of 3 officials
	as, err := sss.VotingScenario(
		"auditors", 1, 2,
		"officials", 2, 3,
		secret,
	)
	if err != nil {
		t.Fatalf("VotingScenario() failed: %v", err)
	}

	// Generate shares
	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Verify share counts
	if len(allShares["auditors"]) != 2 {
		t.Errorf("Expected 2 auditor shares, got %d", len(allShares["auditors"]))
	}
	if len(allShares["officials"]) != 3 {
		t.Errorf("Expected 3 official shares, got %d", len(allShares["officials"]))
	}

	// Test reconstruction with 1 auditor + 2 officials (should work)
	t.Run("valid: 1 auditor + 2 officials", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"auditors":  allShares["auditors"][:1],
			"officials": allShares["officials"][:2],
		}

		if !as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned false for valid share set")
		}

		reconstructed, err := as.ReconstructSecret(providedShares)
		if err != nil {
			t.Errorf("ReconstructSecret() failed: %v", err)
			return
		}

		originalInt := new(big.Int).SetBytes(secret)
		originalInt.Mod(originalInt, sss.Prime)
		reconstructedInt := new(big.Int).SetBytes(reconstructed)

		if originalInt.Cmp(reconstructedInt) != 0 {
			t.Error("Reconstructed secret does not match original")
		}
	})

	// Test with only auditor (should fail)
	t.Run("invalid: only auditors", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"auditors": allShares["auditors"][:1],
		}

		if as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned true without officials")
		}

		_, err := as.ReconstructSecret(providedShares)
		if err == nil {
			t.Error("ReconstructSecret() should fail without officials")
		}
	})

	// Test with only officials (should fail)
	t.Run("invalid: only officials", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"officials": allShares["officials"][:2],
		}

		if as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned true without auditor")
		}

		_, err := as.ReconstructSecret(providedShares)
		if err == nil {
			t.Error("ReconstructSecret() should fail without auditor")
		}
	})

	// Test with insufficient officials (should fail)
	t.Run("invalid: auditor + 1 official", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"auditors":  allShares["auditors"][:1],
			"officials": allShares["officials"][:1],
		}

		if as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned true with insufficient officials")
		}

		_, err := as.ReconstructSecret(providedShares)
		if err == nil {
			t.Error("ReconstructSecret() should fail with insufficient officials")
		}
	})
}

// TestAccessStructure_ANDNode tests AND access structure.
func TestAccessStructure_ANDNode(t *testing.T) {
	secret := []byte("AND test secret")
	as := sss.NewAccessStructure(secret)

	// Create two groups that must both participate
	if err := as.AddGroup("group_a", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("group_b", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}

	// Set access tree: group_a AND group_b
	tree := sss.AND(sss.Leaf("group_a"), sss.Leaf("group_b"))
	if err := as.SetAccessTree(tree); err != nil {
		t.Fatalf("SetAccessTree() failed: %v", err)
	}

	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Both groups required
	t.Run("both groups present", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"group_a": allShares["group_a"][:1],
			"group_b": allShares["group_b"][:1],
		}

		if !as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned false for valid AND combination")
		}

		reconstructed, err := as.ReconstructSecret(providedShares)
		if err != nil {
			t.Errorf("ReconstructSecret() failed: %v", err)
			return
		}

		originalInt := new(big.Int).SetBytes(secret)
		originalInt.Mod(originalInt, sss.Prime)
		reconstructedInt := new(big.Int).SetBytes(reconstructed)

		if originalInt.Cmp(reconstructedInt) != 0 {
			t.Error("Reconstructed secret does not match original")
		}
	})

	// Only group_a (should fail)
	t.Run("only group_a", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"group_a": allShares["group_a"][:1],
		}

		if as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned true with only group_a")
		}
	})
}

// TestAccessStructure_ORNode tests OR access structure.
func TestAccessStructure_ORNode(t *testing.T) {
	secret := []byte("OR test secret")
	as := sss.NewAccessStructure(secret)

	// Create three groups where any 2 can participate
	if err := as.AddGroup("group_a", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("group_b", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("group_c", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}

	// Set access tree: OR(2, group_a, group_b, group_c) - any 2 of 3 groups
	tree := sss.OR(2, sss.Leaf("group_a"), sss.Leaf("group_b"), sss.Leaf("group_c"))
	if err := as.SetAccessTree(tree); err != nil {
		t.Fatalf("SetAccessTree() failed: %v", err)
	}

	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Test combinations
	combinations := []struct {
		name   string
		groups []string
		valid  bool
	}{
		{"a+b", []string{"group_a", "group_b"}, true},
		{"a+c", []string{"group_a", "group_c"}, true},
		{"b+c", []string{"group_b", "group_c"}, true},
		{"all three", []string{"group_a", "group_b", "group_c"}, true},
		{"only a", []string{"group_a"}, false},
		{"only b", []string{"group_b"}, false},
	}

	for _, combo := range combinations {
		t.Run(combo.name, func(t *testing.T) {
			providedShares := make(map[string][]*sss.AccessShare)
			for _, g := range combo.groups {
				providedShares[g] = allShares[g][:1]
			}

			canReconstruct := as.CanReconstruct(providedShares)
			if canReconstruct != combo.valid {
				t.Errorf("CanReconstruct() = %v, want %v", canReconstruct, combo.valid)
			}

			_, err := as.ReconstructSecret(providedShares)
			if combo.valid && err != nil {
				t.Errorf("ReconstructSecret() failed: %v", err)
			}
			if !combo.valid && err == nil {
				t.Error("ReconstructSecret() should have failed")
			}
		})
	}
}

// TestAccessStructure_ComplexTree tests nested AND/OR structures.
func TestAccessStructure_ComplexTree(t *testing.T) {
	secret := []byte("complex tree secret")
	as := sss.NewAccessStructure(secret)

	// Create groups
	if err := as.AddGroup("admin", 1, 1); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("auditor", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("official_a", 1, 1); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}
	if err := as.AddGroup("official_b", 1, 1); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}

	// Complex structure: admin OR (auditor AND (official_a OR official_b))
	tree := sss.OR(1,
		sss.Leaf("admin"),
		sss.AND(
			sss.Leaf("auditor"),
			sss.OR(1, sss.Leaf("official_a"), sss.Leaf("official_b")),
		),
	)
	if err := as.SetAccessTree(tree); err != nil {
		t.Fatalf("SetAccessTree() failed: %v", err)
	}

	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Test: admin alone should work
	t.Run("admin alone", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"admin": allShares["admin"][:1],
		}

		if !as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned false for admin alone")
		}
	})

	// Test: auditor + official_a should work
	t.Run("auditor + official_a", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"auditor":    allShares["auditor"][:1],
			"official_a": allShares["official_a"][:1],
		}

		if !as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned false for auditor + official_a")
		}
	})

	// Test: auditor alone should fail
	t.Run("auditor alone", func(t *testing.T) {
		providedShares := map[string][]*sss.AccessShare{
			"auditor": allShares["auditor"][:1],
		}

		if as.CanReconstruct(providedShares) {
			t.Error("CanReconstruct() returned true for auditor alone")
		}
	})
}

// TestAccessStructure_InvalidTree tests validation of invalid access trees.
func TestAccessStructure_InvalidTree(t *testing.T) {
	as := sss.NewAccessStructure([]byte("test"))

	if err := as.AddGroup("valid_group", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}

	tests := []struct {
		name    string
		tree    *sss.AccessNode
		wantErr bool
	}{
		{
			name:    "nil tree",
			tree:    nil,
			wantErr: true,
		},
		{
			name:    "leaf with unknown group",
			tree:    sss.Leaf("nonexistent"),
			wantErr: true,
		},
		{
			name:    "AND with no children",
			tree:    &sss.AccessNode{Type: 0, Children: nil}, // NodeAND
			wantErr: true,
		},
		{
			name:    "OR with threshold > children",
			tree:    sss.OR(5, sss.Leaf("valid_group")),
			wantErr: true,
		},
		{
			name:    "OR with zero threshold",
			tree:    sss.OR(0, sss.Leaf("valid_group")),
			wantErr: true,
		},
		{
			name:    "valid leaf",
			tree:    sss.Leaf("valid_group"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := as.SetAccessTree(tt.tree)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetAccessTree() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAccessStructure_Commitment tests commitment computation.
func TestAccessStructure_Commitment(t *testing.T) {
	secret := []byte("commitment test secret")

	as, err := sss.VotingScenario(
		"auditors", 1, 2,
		"officials", 2, 3,
		secret,
	)
	if err != nil {
		t.Fatalf("VotingScenario() failed: %v", err)
	}

	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Compute commitment
	commitment := as.ComputeAccessCommitment(allShares)

	if len(commitment) != 32 {
		t.Errorf("ComputeAccessCommitment() length = %d, want 32", len(commitment))
	}

	// Same shares should produce same commitment
	commitment2 := as.ComputeAccessCommitment(allShares)
	if !bytes.Equal(commitment, commitment2) {
		t.Error("ComputeAccessCommitment() not deterministic")
	}
}

// TestVotingScenario_InvalidParameters tests error handling in VotingScenario.
func TestVotingScenario_InvalidParameters(t *testing.T) {
	tests := []struct {
		name             string
		auditorThreshold int
		auditorTotal     int
		officialThreshold int
		officialTotal     int
		wantErr          bool
	}{
		{
			name:              "valid parameters",
			auditorThreshold:  1,
			auditorTotal:      2,
			officialThreshold: 2,
			officialTotal:     3,
			wantErr:           false,
		},
		{
			name:              "invalid auditor threshold",
			auditorThreshold:  5,
			auditorTotal:      2,
			officialThreshold: 2,
			officialTotal:     3,
			wantErr:           true,
		},
		{
			name:              "invalid official threshold",
			auditorThreshold:  1,
			auditorTotal:      2,
			officialThreshold: 5,
			officialTotal:     3,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sss.VotingScenario(
				"auditors", tt.auditorThreshold, tt.auditorTotal,
				"officials", tt.officialThreshold, tt.officialTotal,
				[]byte("test secret"),
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("VotingScenario() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAccessStructure_EmptySecret tests handling of empty secrets.
func TestAccessStructure_EmptySecret(t *testing.T) {
	as := sss.NewAccessStructure([]byte{})

	if err := as.AddGroup("group", 1, 2); err != nil {
		t.Fatalf("AddGroup() failed: %v", err)
	}

	if err := as.SetAccessTree(sss.Leaf("group")); err != nil {
		t.Fatalf("SetAccessTree() failed: %v", err)
	}

	_, err := as.GenerateShares()
	if err != sss.ErrEmptySecret {
		t.Errorf("GenerateShares() with empty secret: error = %v, want %v", err, sss.ErrEmptySecret)
	}
}

// TestAccessStructure_RealisticVotingWorkflow simulates a real voting reveal workflow.
func TestAccessStructure_RealisticVotingWorkflow(t *testing.T) {
	// Scenario: Election system where ballots can only be revealed when:
	// - At least 1 auditor agrees (from a pool of 2)
	// - At least 2 election officials agree (from a pool of 3)
	
	masterKey := []byte("AES-256-GCM-KEY-FOR-BALLOT-ENCRYPTION")

	// Step 1: Setup phase - create access structure
	as, err := sss.VotingScenario(
		"auditors", 1, 2,
		"election_officials", 2, 3,
		masterKey,
	)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Step 2: Share distribution - generate and distribute shares
	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("Share generation failed: %v", err)
	}

	// Step 3: Commitment phase - publish commitment to freeze ballots
	commitment := as.ComputeAccessCommitment(allShares)
	t.Logf("Published commitment: %x", commitment)

	// Simulate shares being held by different parties:
	// - Auditor Alice: auditor share 0
	// - Auditor Bob: auditor share 1
	// - Official Carol: official share 0
	// - Official Dave: official share 1
	// - Official Eve: official share 2

	auditorAlice := allShares["auditors"][0]
	// auditorBob := allShares["auditors"][1]
	officialCarol := allShares["election_officials"][0]
	officialDave := allShares["election_officials"][1]
	// officialEve := allShares["election_officials"][2]

	// Step 4: Reveal phase - collect shares and reconstruct
	// Alice (auditor) + Carol and Dave (officials) agree to reveal
	revealShares := map[string][]*sss.AccessShare{
		"auditors":           {auditorAlice},
		"election_officials": {officialCarol, officialDave},
	}

	// Verify access is possible before reconstruction
	if !as.CanReconstruct(revealShares) {
		t.Fatal("CanReconstruct() returned false - access denied before reveal")
	}

	// Reconstruct the master key
	reconstructedKey, err := as.ReconstructSecret(revealShares)
	if err != nil {
		t.Fatalf("Secret reconstruction failed: %v", err)
	}

	// Verify the reconstructed key matches (accounting for modular reduction)
	originalInt := new(big.Int).SetBytes(masterKey)
	originalInt.Mod(originalInt, sss.Prime)
	reconstructedInt := new(big.Int).SetBytes(reconstructedKey)

	if originalInt.Cmp(reconstructedInt) != 0 {
		t.Error("Reconstructed key does not match original master key")
	}

	t.Log("Voting reveal workflow completed successfully")
}

// TestAccessCommitmentDeterministic tests that commitments are deterministic regardless of map iteration order.
func TestAccessCommitmentDeterministic(t *testing.T) {
	secret := []byte("deterministic commitment test")

	as, err := sss.VotingScenario(
		"auditors", 1, 2,
		"officials", 2, 3,
		secret,
	)
	if err != nil {
		t.Fatalf("VotingScenario() failed: %v", err)
	}

	allShares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Compute commitment multiple times
	commitments := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		commitments[i] = as.ComputeAccessCommitment(allShares)
	}

	// All commitments should be identical
	firstCommitment := commitments[0]
	for i, commitment := range commitments[1:] {
		if !bytes.Equal(firstCommitment, commitment) {
			t.Errorf("Commitment %d differs from first commitment", i+1)
		}
	}
}

// TestAccessCommitmentDifferentSharesProduceDifferentCommitments tests that
// different share sets produce different commitments.
func TestAccessCommitmentDifferentSharesProduceDifferentCommitments(t *testing.T) {
	secret := []byte("different shares test")

	// Create two access structures with same configuration but different shares
	as1, _ := sss.VotingScenario("auditors", 1, 2, "officials", 2, 3, secret)
	as2, _ := sss.VotingScenario("auditors", 1, 2, "officials", 2, 3, secret)

	shares1, _ := as1.GenerateShares()
	shares2, _ := as2.GenerateShares()

	commitment1 := as1.ComputeAccessCommitment(shares1)
	commitment2 := as2.ComputeAccessCommitment(shares2)

	// Different random shares should produce different commitments
	if bytes.Equal(commitment1, commitment2) {
		t.Error("Different share sets should produce different commitments")
	}
}

// TestAccessCommitmentGroupOrdering tests that group ordering doesn't affect determinism.
func TestAccessCommitmentGroupOrdering(t *testing.T) {
	secret := []byte("group ordering test")

	as := sss.NewAccessStructure(secret)

	// Add groups in different order
	as.AddGroup("zebra_group", 1, 2)
	as.AddGroup("alpha_group", 1, 2)
	as.AddGroup("middle_group", 1, 2)

	// Set up OR structure requiring any 2 of 3 groups
	as.SetAccessTree(sss.OR(2,
		sss.Leaf("zebra_group"),
		sss.Leaf("alpha_group"),
		sss.Leaf("middle_group"),
	))

	shares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Compute commitment multiple times
	commitments := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		commitments[i] = as.ComputeAccessCommitment(shares)
	}

	// All should be identical due to sorted group names
	for i := 1; i < len(commitments); i++ {
		if !bytes.Equal(commitments[0], commitments[i]) {
			t.Errorf("Commitment %d differs despite same shares", i)
		}
	}
}

// TestAccessStructureNodeTypes tests the helper functions for creating nodes.
func TestAccessStructureNodeTypes(t *testing.T) {
	// Test AND node creation
	andNode := sss.AND(sss.Leaf("a"), sss.Leaf("b"))
	if andNode == nil {
		t.Fatal("AND() returned nil")
	}
	if len(andNode.Children) != 2 {
		t.Errorf("AND() children count = %d, want 2", len(andNode.Children))
	}

	// Test OR node creation
	orNode := sss.OR(2, sss.Leaf("a"), sss.Leaf("b"), sss.Leaf("c"))
	if orNode == nil {
		t.Fatal("OR() returned nil")
	}
	if len(orNode.Children) != 3 {
		t.Errorf("OR() children count = %d, want 3", len(orNode.Children))
	}
	if orNode.Threshold != 2 {
		t.Errorf("OR() threshold = %d, want 2", orNode.Threshold)
	}

	// Test Leaf node creation
	leafNode := sss.Leaf("test_group")
	if leafNode == nil {
		t.Fatal("Leaf() returned nil")
	}
	if leafNode.Group != "test_group" {
		t.Errorf("Leaf() group = %s, want test_group", leafNode.Group)
	}
}

// TestAccessStructureDeepNesting tests deeply nested access structures.
func TestAccessStructureDeepNesting(t *testing.T) {
	secret := []byte("deep nesting test")
	as := sss.NewAccessStructure(secret)

	as.AddGroup("level1", 1, 1)
	as.AddGroup("level2a", 1, 1)
	as.AddGroup("level2b", 1, 1)
	as.AddGroup("level3", 1, 1)

	// Create deeply nested structure:
	// AND(level1, OR(1, AND(level2a, level3), level2b))
	tree := sss.AND(
		sss.Leaf("level1"),
		sss.OR(1,
			sss.AND(sss.Leaf("level2a"), sss.Leaf("level3")),
			sss.Leaf("level2b"),
		),
	)

	err := as.SetAccessTree(tree)
	if err != nil {
		t.Fatalf("SetAccessTree() failed: %v", err)
	}

	shares, err := as.GenerateShares()
	if err != nil {
		t.Fatalf("GenerateShares() failed: %v", err)
	}

	// Test that we can check reconstruction
	// With level1 + level2b (satisfies OR via second branch)
	validShares := map[string][]*sss.AccessShare{
		"level1":  shares["level1"][:1],
		"level2b": shares["level2b"][:1],
	}

	if !as.CanReconstruct(validShares) {
		t.Error("CanReconstruct() should return true for valid share combination")
	}

	// With only level1 (missing OR branch)
	invalidShares := map[string][]*sss.AccessShare{
		"level1": shares["level1"][:1],
	}

	if as.CanReconstruct(invalidShares) {
		t.Error("CanReconstruct() should return false without OR branch satisfied")
	}
}
