// Package types defines shared data types used across the application.
package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ObjectId = primitive.ObjectID

// UserRole represents the role of a user in the system.
type UserRole string

const (
	// RoleVoter is a regular user who can vote in polls.
	RoleVoter UserRole = "voter"
	// RoleAuditor is a user who can audit polls and participate in secret reveal.
	RoleAuditor UserRole = "auditor"
	// RoleOfficial is an election official who can create polls and reveal secrets.
	RoleOfficial UserRole = "official"
	// RoleAdmin is a system administrator with full access.
	RoleAdmin UserRole = "admin"
)

// User represents a user in the database.
type User struct {
	ID       ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string   `json:"username" bson:"username"`
	Password string   `json:"-" bson:"password"`
	Salt     string   `json:"-" bson:"salt"`
	Role     UserRole `json:"role" bson:"role"`
	Date     string   `json:"date" bson:"date"`
	Version  int      `json:"version" bson:"version"`
}

// AccessStructureType represents the type of access structure for revealing poll results.
type AccessStructureType string

const (
	// AccessStructureOfficialsOnly requires only officials to contribute shares.
	AccessStructureOfficialsOnly AccessStructureType = "officials_only"
	// AccessStructureAuditorsOnly requires only auditors to contribute shares.
	AccessStructureAuditorsOnly AccessStructureType = "auditors_only"
	// AccessStructureBoth requires both officials AND auditors to contribute shares.
	AccessStructureBoth AccessStructureType = "both"
)

// PollStatus represents the current status of a poll.
type PollStatus string

const (
	// PollStatusDraft indicates the poll is being created and not yet open.
	PollStatusDraft PollStatus = "draft"
	// PollStatusOpen indicates the poll is open for voting.
	PollStatusOpen PollStatus = "open"
	// PollStatusClosed indicates voting has ended, shares have been distributed, and results reveal is pending.
	PollStatusClosed PollStatus = "closed"
	// PollStatusRevealed indicates results have been revealed via secret sharing.
	PollStatusRevealed PollStatus = "revealed"
)

// PollOption represents a voting option in a poll.
type PollOption struct {
	ID    string `json:"id" bson:"id" example:"option_1"`
	Label string `json:"label" bson:"label" example:"Candidate A"`
}

// Poll represents a voting poll/election scenario.
type Poll struct {
	ID          ObjectId     `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string       `json:"title" bson:"title"`
	Description string       `json:"description" bson:"description"`
	CreatorID   ObjectId     `json:"creator_id" bson:"creator_id"`
	Options     []PollOption `json:"options" bson:"options"`
	Status      PollStatus   `json:"status" bson:"status"`
	StartTime   *time.Time   `json:"start_time,omitempty" bson:"start_time,omitempty"`
	EndTime     *time.Time   `json:"end_time,omitempty" bson:"end_time,omitempty"`
	// Access structure configuration for secret sharing
	AccessStructureType  AccessStructureType `json:"access_structure_type" bson:"access_structure_type"`
	MinAuditorsRequired  int                 `json:"min_auditors_required" bson:"min_auditors_required"`
	MinOfficialsRequired int                 `json:"min_officials_required" bson:"min_officials_required"`
	TotalAuditors        int                 `json:"total_auditors" bson:"total_auditors"`
	TotalOfficials       int                 `json:"total_officials" bson:"total_officials"`
	// Commitment hash published when ballots are frozen
	BallotCommitment string `json:"ballot_commitment,omitempty" bson:"ballot_commitment,omitempty"`
	CreatedAt        string `json:"created_at" bson:"created_at"`
	UpdatedAt        string `json:"updated_at" bson:"updated_at"`
	Version          int    `json:"version" bson:"version"`
}

// EncryptedBallot represents an encrypted vote in the system.
type EncryptedBallot struct {
	ID             ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	PollID         ObjectId `json:"poll_id" bson:"poll_id"`
	VoterID        ObjectId `json:"voter_id" bson:"voter_id"`
	EncryptedVote  string   `json:"encrypted_vote" bson:"encrypted_vote"`
	VoteCommitment string   `json:"vote_commitment" bson:"vote_commitment"`
	CastAt         string   `json:"cast_at" bson:"cast_at"`
	Version        int      `json:"version" bson:"version"`
}

// SecretShare represents a share of the poll encryption key held by a participant.
type SecretShare struct {
	ID            ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	PollID        ObjectId `json:"poll_id" bson:"poll_id"`
	HolderID      ObjectId `json:"holder_id" bson:"holder_id"`
	GroupName     string   `json:"group_name" bson:"group_name"`
	ShareIndex    int      `json:"share_index" bson:"share_index"`
	ShareValue    string   `json:"share_value" bson:"share_value"`
	Commitment    string   `json:"commitment" bson:"commitment"`
	IsContributed bool     `json:"is_contributed" bson:"is_contributed"`
	CreatedAt     string   `json:"created_at" bson:"created_at"`
	Version       int      `json:"version" bson:"version"`
}

// PollResult represents the revealed results of a poll.
type PollResult struct {
	ID         ObjectId         `json:"_id,omitempty" bson:"_id,omitempty"`
	PollID     ObjectId         `json:"poll_id" bson:"poll_id"`
	Results    map[string]int64 `json:"results" bson:"results"`
	TotalVotes int64            `json:"total_votes" bson:"total_votes"`
	RevealedAt string           `json:"revealed_at" bson:"revealed_at"`
	RevealedBy []ObjectId       `json:"revealed_by" bson:"revealed_by"`
	Version    int              `json:"version" bson:"version"`
}

// ===============================
// Request Types
// ===============================

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Username string `json:"username" bson:"username" binding:"required" example:"johndoe"`
	Password string `json:"password" binding:"required" example:"secretpassword"`
}

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Username string   `json:"username" binding:"required" example:"johndoe"`
	Password string   `json:"password" binding:"required" example:"secretpassword"`
	Role     UserRole `json:"role" binding:"required" example:"voter"`
}

// CreatePollRequest represents the request body for creating a new poll.
type CreatePollRequest struct {
	Title                string              `json:"title" binding:"required" example:"Board Election 2024"`
	Description          string              `json:"description" example:"Annual board member election"`
	Options              []PollOption        `json:"options" binding:"required,min=2"`
	StartTime            *time.Time          `json:"start_time,omitempty"`
	EndTime              *time.Time          `json:"end_time,omitempty"`
	AccessStructureType  AccessStructureType `json:"access_structure_type" binding:"required" example:"both"`
	MinAuditorsRequired  int                 `json:"min_auditors_required" binding:"min=0" example:"1"`
	MinOfficialsRequired int                 `json:"min_officials_required" binding:"min=0" example:"2"`
}

// UpdatePollRequest represents the request body for updating a poll.
type UpdatePollRequest struct {
	Title       string       `json:"title,omitempty" example:"Updated Board Election 2024"`
	Description string       `json:"description,omitempty" example:"Updated description"`
	Options     []PollOption `json:"options,omitempty"`
	StartTime   *time.Time   `json:"start_time,omitempty"`
	EndTime     *time.Time   `json:"end_time,omitempty"`
}

// UpdatePollStatusRequest represents the request to change poll status.
type UpdatePollStatusRequest struct {
	Status PollStatus `json:"status" binding:"required" example:"open"`
}

// CastBallotRequest represents the request body for casting a vote.
type CastBallotRequest struct {
	PollID        string `json:"poll_id" binding:"required" example:"507f1f77bcf86cd799439011"`
	EncryptedVote string `json:"encrypted_vote" binding:"required" example:"base64_encrypted_data"`
}

// ContributeShareRequest represents a request to contribute a share for reveal.
type ContributeShareRequest struct {
	PollID     string `json:"poll_id" binding:"required" example:"507f1f77bcf86cd799439011"`
	ShareValue string `json:"share_value" binding:"required" example:"base64_share_value"`
}

// SSSTestRequest represents a request to test the secret sharing algorithm.
type SSSTestRequest struct {
	Secret    string `json:"secret" binding:"required" example:"my_test_secret"`
	Threshold int    `json:"threshold" binding:"required,min=1" example:"2"`
	Total     int    `json:"total" binding:"required,min=1" example:"3"`
}

// ===============================
// Response Types
// ===============================

// AuthResponse represents the response for successful authentication.
type AuthResponse struct {
	UserID   string   `json:"user_id" example:"507f1f77bcf86cd799439011"`
	Token    string   `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Role     UserRole `json:"role" example:"voter"`
	Username string   `json:"username" example:"johndoe"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid credentials"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

// PollResponse represents a poll in API responses.
type PollResponse struct {
	ID                   string              `json:"id" example:"507f1f77bcf86cd799439011"`
	Title                string              `json:"title" example:"Board Election 2024"`
	Description          string              `json:"description" example:"Annual board member election"`
	CreatorID            string              `json:"creator_id" example:"507f1f77bcf86cd799439011"`
	Options              []PollOption        `json:"options"`
	Status               PollStatus          `json:"status" example:"open"`
	StartTime            *time.Time          `json:"start_time,omitempty"`
	EndTime              *time.Time          `json:"end_time,omitempty"`
	AccessStructureType  AccessStructureType `json:"access_structure_type" example:"both"`
	MinAuditorsRequired  int                 `json:"min_auditors_required" example:"1"`
	MinOfficialsRequired int                 `json:"min_officials_required" example:"2"`
	TotalAuditors        int                 `json:"total_auditors" example:"2"`
	TotalOfficials       int                 `json:"total_officials" example:"3"`
	BallotCommitment     string              `json:"ballot_commitment,omitempty" example:"a1b2c3d4..."`
	CreatedAt            string              `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt            string              `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// PollListResponse represents a list of polls.
type PollListResponse struct {
	Polls []PollResponse `json:"polls"`
	Total int            `json:"total" example:"10"`
}

// BallotResponse represents a ballot receipt.
type BallotResponse struct {
	ID             string `json:"id" example:"507f1f77bcf86cd799439011"`
	PollID         string `json:"poll_id" example:"507f1f77bcf86cd799439011"`
	VoteCommitment string `json:"vote_commitment" example:"a1b2c3d4..."`
	CastAt         string `json:"cast_at" example:"2024-01-15T10:30:00Z"`
}

// ShareDistributionResponse represents shares distributed to participants.
type ShareDistributionResponse struct {
	PollID     string `json:"poll_id" example:"507f1f77bcf86cd799439011"`
	GroupName  string `json:"group_name" example:"auditors"`
	ShareIndex int    `json:"share_index" example:"1"`
	ShareValue string `json:"share_value" example:"base64_share_value"`
	Commitment string `json:"commitment" example:"a1b2c3d4..."`
}

// PollResultResponse represents revealed poll results.
type PollResultResponse struct {
	PollID     string           `json:"poll_id" example:"507f1f77bcf86cd799439011"`
	Results    map[string]int64 `json:"results"`
	TotalVotes int64            `json:"total_votes" example:"150"`
	RevealedAt string           `json:"revealed_at" example:"2024-01-20T15:00:00Z"`
}

// SSSTestResponse represents the response from testing the SSS algorithm.
type SSSTestResponse struct {
	OriginalSecret      string            `json:"original_secret" example:"my_test_secret"`
	ReconstructedSecret string            `json:"reconstructed_secret" example:"my_test_secret"`
	Match               bool              `json:"match" example:"true"`
	Threshold           int               `json:"threshold" example:"2"`
	TotalShares         int               `json:"total_shares" example:"3"`
	ShareCommitments    []string          `json:"share_commitments"`
	UsedShareIndices    []int             `json:"used_share_indices" example:"1,2"`
	AccessStructureTest *AccessTestResult `json:"access_structure_test,omitempty"`
}

// AccessTestResult represents results from testing access structures.
type AccessTestResult struct {
	AuditorGroup   string `json:"auditor_group" example:"auditors"`
	OfficialGroup  string `json:"official_group" example:"officials"`
	TreeStructure  string `json:"tree_structure" example:"AND(auditors, officials)"`
	CanReconstruct bool   `json:"can_reconstruct" example:"true"`
	Message        string `json:"message" example:"Access structure validated successfully"`
}

// UserResponse represents a user in API responses (without sensitive data).
type UserResponse struct {
	ID       string   `json:"id" example:"507f1f77bcf86cd799439011"`
	Username string   `json:"username" example:"johndoe"`
	Role     UserRole `json:"role" example:"voter"`
	Date     string   `json:"date" example:"2024-01-15T10:30:00Z"`
}

// ShareStatusResponse represents the status of share collection for a poll.
type ShareStatusResponse struct {
	PollID               string   `json:"poll_id" example:"507f1f77bcf86cd799439011"`
	AuditorShares        int      `json:"auditor_shares" example:"1"`
	MinAuditorsRequired  int      `json:"min_auditors_required" example:"1"`
	OfficialShares       int      `json:"official_shares" example:"2"`
	MinOfficialsRequired int      `json:"min_officials_required" example:"2"`
	CanReveal            bool     `json:"can_reveal" example:"true"`
	ContributedBy        []string `json:"contributed_by"`
}
