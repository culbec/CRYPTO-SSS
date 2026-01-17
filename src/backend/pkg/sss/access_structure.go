// Package sss provides access structure support for Shamir's Secret Sharing.
//
// Access structures allow defining complex authorization policies beyond
// simple (t,n) thresholds. For example: "1 auditor AND 2 of 3 election officials"
// means the secret can only be reconstructed when an auditor participates
// together with at least 2 election officials.
package sss

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"sort"
)

var (
	// ErrAccessDenied is returned when access structure requirements are not met.
	ErrAccessDenied = errors.New("sss: access structure requirements not satisfied")
	// ErrInvalidAccessStructure is returned when the access structure is malformed.
	ErrInvalidAccessStructure = errors.New("sss: invalid access structure definition")
	// ErrParticipantNotFound is returned when a required participant is missing.
	ErrParticipantNotFound = errors.New("sss: required participant not found")
	// ErrGroupNotFound is returned when a referenced group doesn't exist.
	ErrGroupNotFound = errors.New("sss: participant group not found")
)

// ParticipantGroup represents a group of participants with a threshold requirement.
// For example: "election_officials" group requiring 2 of 3 to participate.
type ParticipantGroup struct {
	Name       string   // Unique identifier for the group
	Threshold  int      // Minimum participants required from this group
	Total      int      // Total participants in this group
	ShareSet   *ShareSet // The shares for this group
}

// AccessNode represents a node in the access structure tree.
// It can be either a leaf (group reference) or an internal node (AND/OR).
type AccessNode struct {
	Type      NodeType      // Type of node: AND, OR, or LEAF
	Group     string        // Group name (only for LEAF nodes)
	Threshold int           // For OR nodes: minimum children required
	Children  []*AccessNode // Child nodes (for AND/OR)
}

// NodeType represents the type of access structure node.
type NodeType int

const (
	// NodeAND requires all children to be satisfied.
	NodeAND NodeType = iota
	// NodeOR requires at least Threshold children to be satisfied.
	NodeOR
	// NodeLEAF represents a participant group.
	NodeLEAF
)

// AccessStructure defines the complete access control policy.
type AccessStructure struct {
	Groups     map[string]*ParticipantGroup // All participant groups
	Root       *AccessNode                  // Root of the access tree
	MasterKey  []byte                       // The master secret being protected
	GroupKeys  map[string][]byte            // Intermediate keys for each group
}

// AccessShare represents a share held by a participant in a specific group.
type AccessShare struct {
	GroupName string   // The group this share belongs to
	Share     *Share   // The actual SSS share
}

// NewAccessStructure creates a new access structure with the specified master secret.
func NewAccessStructure(masterSecret []byte) *AccessStructure {
	return &AccessStructure{
		Groups:    make(map[string]*ParticipantGroup),
		GroupKeys: make(map[string][]byte),
		MasterKey: masterSecret,
	}
}

// AddGroup adds a participant group with threshold requirements.
// threshold specifies minimum participants needed from this group.
// total specifies the total number of participants in the group.
func (as *AccessStructure) AddGroup(name string, threshold, total int) error {
	if threshold <= 0 || threshold > total {
		return ErrInvalidThreshold
	}

	as.Groups[name] = &ParticipantGroup{
		Name:      name,
		Threshold: threshold,
		Total:     total,
	}
	return nil
}

// SetAccessTree sets the access tree defining the authorization policy.
// Example: AND(auditor, threshold(2, [official1, official2, official3]))
func (as *AccessStructure) SetAccessTree(root *AccessNode) error {
	if err := as.validateAccessTree(root); err != nil {
		return err
	}
	as.Root = root
	return nil
}

// validateAccessTree validates the access tree structure.
func (as *AccessStructure) validateAccessTree(node *AccessNode) error {
	if node == nil {
		return ErrInvalidAccessStructure
	}

	switch node.Type {
	case NodeLEAF:
		if _, exists := as.Groups[node.Group]; !exists {
			return fmt.Errorf("%w: %s", ErrGroupNotFound, node.Group)
		}
	case NodeAND:
		if len(node.Children) == 0 {
			return ErrInvalidAccessStructure
		}
		for _, child := range node.Children {
			if err := as.validateAccessTree(child); err != nil {
				return err
			}
		}
	case NodeOR:
		if len(node.Children) == 0 || node.Threshold <= 0 || node.Threshold > len(node.Children) {
			return ErrInvalidAccessStructure
		}
		for _, child := range node.Children {
			if err := as.validateAccessTree(child); err != nil {
				return err
			}
		}
	default:
		return ErrInvalidAccessStructure
	}

	return nil
}

// GenerateShares generates all shares according to the access structure.
// It uses hierarchical secret sharing where the master secret is split
// according to the access tree structure.
func (as *AccessStructure) GenerateShares() (map[string][]*AccessShare, error) {
	if as.Root == nil {
		return nil, ErrInvalidAccessStructure
	}
	if len(as.MasterKey) == 0 {
		return nil, ErrEmptySecret
	}

	result := make(map[string][]*AccessShare)

	// Generate shares based on tree structure
	if err := as.generateNodeShares(as.Root, as.MasterKey, result); err != nil {
		return nil, err
	}

	return result, nil
}

// generateNodeShares recursively generates shares for each node in the tree.
func (as *AccessStructure) generateNodeShares(node *AccessNode, secret []byte, result map[string][]*AccessShare) error {
	switch node.Type {
	case NodeLEAF:
		// For leaf nodes, generate shares for the group
		group := as.Groups[node.Group]
		shareSet, err := Split(secret, group.Threshold, group.Total)
		if err != nil {
			return err
		}
		group.ShareSet = shareSet
		as.GroupKeys[node.Group] = secret

		// Convert to AccessShares
		shares := make([]*AccessShare, len(shareSet.Shares))
		for i, share := range shareSet.Shares {
			shares[i] = &AccessShare{
				GroupName: node.Group,
				Share:     share,
			}
		}
		result[node.Group] = shares

	case NodeAND:
		// For AND nodes, all children share the same secret
		// Each child gets the same secret to reconstruct
		for _, child := range node.Children {
			if err := as.generateNodeShares(child, secret, result); err != nil {
				return err
			}
		}

	case NodeOR:
		// For OR nodes, use threshold sharing on the secret
		// Split the secret into shares, one for each child
		shareSet, err := Split(secret, node.Threshold, len(node.Children))
		if err != nil {
			return err
		}

		// Each child gets a share as their "secret" to further split
		for i, child := range node.Children {
			childSecret := shareSet.Shares[i].Value.Bytes()
			if err := as.generateNodeShares(child, childSecret, result); err != nil {
				return err
			}
		}
	}

	return nil
}

// ReconstructSecret attempts to reconstruct the master secret from provided shares.
// It returns the secret only if the access structure requirements are satisfied.
func (as *AccessStructure) ReconstructSecret(providedShares map[string][]*AccessShare) ([]byte, error) {
	if as.Root == nil {
		return nil, ErrInvalidAccessStructure
	}

	return as.reconstructFromNode(as.Root, providedShares)
}

// reconstructFromNode recursively reconstructs secrets according to the tree.
func (as *AccessStructure) reconstructFromNode(node *AccessNode, providedShares map[string][]*AccessShare) ([]byte, error) {
	switch node.Type {
	case NodeLEAF:
		group := as.Groups[node.Group]
		shares, exists := providedShares[node.Group]
		if !exists || len(shares) < group.Threshold {
			return nil, fmt.Errorf("%w: group %s needs %d shares, got %d",
				ErrAccessDenied, node.Group, group.Threshold, len(shares))
		}

		// Extract the Share objects
		rawShares := make([]*Share, len(shares))
		for i, s := range shares {
			rawShares[i] = s.Share
		}

		return Combine(rawShares, group.Threshold)

	case NodeAND:
		// All children must provide the same reconstructed value
		var result []byte
		for _, child := range node.Children {
			childResult, err := as.reconstructFromNode(child, providedShares)
			if err != nil {
				return nil, err
			}
			if result == nil {
				result = childResult
			}
			// In AND structure, all children reconstruct the same secret
		}
		return result, nil

	case NodeOR:
		// Need threshold number of children to succeed
		var reconstructedShares []*Share
		shareIndex := 1

		for _, child := range node.Children {
			childResult, err := as.reconstructFromNode(child, providedShares)
			if err == nil {
				reconstructedShares = append(reconstructedShares, &Share{
					Index: shareIndex,
					Value: bigIntFromBytes(childResult),
				})
			}
			shareIndex++

			if len(reconstructedShares) >= node.Threshold {
				break
			}
		}

		if len(reconstructedShares) < node.Threshold {
			return nil, fmt.Errorf("%w: OR node needs %d children satisfied, got %d",
				ErrAccessDenied, node.Threshold, len(reconstructedShares))
		}

		return Combine(reconstructedShares, node.Threshold)

	default:
		return nil, ErrInvalidAccessStructure
	}
}

// bigIntFromBytes converts bytes to big.Int safely.
func bigIntFromBytes(b []byte) *big.Int {
	if len(b) == 0 {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(b)
}

// CanReconstruct checks if the provided shares satisfy the access structure
// without actually performing reconstruction.
func (as *AccessStructure) CanReconstruct(providedShares map[string][]*AccessShare) bool {
	if as.Root == nil {
		return false
	}
	return as.canReconstructFromNode(as.Root, providedShares)
}

// canReconstructFromNode checks if a node's requirements are satisfied.
func (as *AccessStructure) canReconstructFromNode(node *AccessNode, providedShares map[string][]*AccessShare) bool {
	switch node.Type {
	case NodeLEAF:
		group := as.Groups[node.Group]
		shares, exists := providedShares[node.Group]
		return exists && len(shares) >= group.Threshold

	case NodeAND:
		for _, child := range node.Children {
			if !as.canReconstructFromNode(child, providedShares) {
				return false
			}
		}
		return true

	case NodeOR:
		satisfied := 0
		for _, child := range node.Children {
			if as.canReconstructFromNode(child, providedShares) {
				satisfied++
			}
		}
		return satisfied >= node.Threshold

	default:
		return false
	}
}

// AND creates an AND access node requiring all children to be satisfied.
func AND(children ...*AccessNode) *AccessNode {
	return &AccessNode{
		Type:     NodeAND,
		Children: children,
	}
}

// OR creates an OR access node requiring at least threshold children.
func OR(threshold int, children ...*AccessNode) *AccessNode {
	return &AccessNode{
		Type:      NodeOR,
		Threshold: threshold,
		Children:  children,
	}
}

// Leaf creates a leaf node referencing a participant group.
func Leaf(groupName string) *AccessNode {
	return &AccessNode{
		Type:  NodeLEAF,
		Group: groupName,
	}
}

// ComputeAccessCommitment computes a SHA-256 commitment of all shares.
// This can be published to freeze the shares before reveal.
// Group names are sorted for deterministic commitment.
func (as *AccessStructure) ComputeAccessCommitment(allShares map[string][]*AccessShare) []byte {
	h := sha256.New()

	// Sort group names for deterministic commitment
	groupNames := make([]string, 0, len(allShares))
	for groupName := range allShares {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)

	for _, groupName := range groupNames {
		shares := allShares[groupName]
		h.Write([]byte(groupName))
		for _, share := range shares {
			h.Write(ComputeShareCommitment(share.Share))
		}
	}

	return h.Sum(nil)
}

// VotingScenario creates a common voting access structure:
// "threshold auditors AND threshold officials" must agree to reveal.
//
// For example: VotingScenario("auditors", 1, 2, "officials", 2, 3, secret)
// creates a structure where 1 of 2 auditors AND 2 of 3 officials must participate.
func VotingScenario(
	auditorGroup string, auditorThreshold, auditorTotal int,
	officialGroup string, officialThreshold, officialTotal int,
	secret []byte,
) (*AccessStructure, error) {
	as := NewAccessStructure(secret)

	if err := as.AddGroup(auditorGroup, auditorThreshold, auditorTotal); err != nil {
		return nil, err
	}
	if err := as.AddGroup(officialGroup, officialThreshold, officialTotal); err != nil {
		return nil, err
	}

	// Access tree: auditorGroup AND officialGroup
	tree := AND(Leaf(auditorGroup), Leaf(officialGroup))
	if err := as.SetAccessTree(tree); err != nil {
		return nil, err
	}

	return as, nil
}
