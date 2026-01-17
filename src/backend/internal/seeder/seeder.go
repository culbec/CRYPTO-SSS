// Package seeder provides functionality to seed the database with sample data.
package seeder

import (
	"context"
	"encoding/hex"
	"time"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/security"
	"go.mongodb.org/mongo-driver/bson"
)

// Seeder handles database seeding operations.
type Seeder struct {
	db     *mongo.Client
	hasher *security.Argon2idHash
}

// NewSeeder creates a new Seeder instance.
func NewSeeder(db *mongo.Client) *Seeder {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)
	return &Seeder{db: db, hasher: hasher}
}

// SeedAll seeds all sample data if the database is empty.
func (s *Seeder) SeedAll(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	logger.Info("Checking if database needs seeding...")

	// Check if users already exist
	var existingUsers []types.User
	_, err := s.db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{},
		nil,
		&existingUsers,
	)
	if err == nil && len(existingUsers) > 0 {
		logger.Info("Database already has data, skipping seeding", "users", len(existingUsers))
		return nil
	}

	logger.Info("Seeding database with sample data...")

	// Seed users
	if err := s.seedUsers(ctx); err != nil {
		logger.Error("Failed to seed users", "error", err)
		return err
	}

	// Seed polls
	if err := s.seedPolls(ctx); err != nil {
		logger.Error("Failed to seed polls", "error", err)
		return err
	}

	logger.Info("Database seeding completed successfully!")
	return nil
}

// seedUsers creates sample users with different roles.
func (s *Seeder) seedUsers(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	users := []struct {
		username string
		password string
		role     types.UserRole
	}{
		// Admin users
		{"admin", "admin123", types.RoleAdmin},
		// Auditors
		{"auditor1", "auditor123", types.RoleAuditor},
		{"auditor2", "auditor123", types.RoleAuditor},
		{"auditor3", "auditor123", types.RoleAuditor},
		// Officials
		{"official1", "official123", types.RoleOfficial},
		{"official2", "official123", types.RoleOfficial},
		{"official3", "official123", types.RoleOfficial},
		{"official4", "official123", types.RoleOfficial},
		// Voters
		{"voter1", "voter123", types.RoleVoter},
		{"voter2", "voter123", types.RoleVoter},
		{"voter3", "voter123", types.RoleVoter},
		{"voter4", "voter123", types.RoleVoter},
		{"voter5", "voter123", types.RoleVoter},
		{"voter6", "voter123", types.RoleVoter},
		{"voter7", "voter123", types.RoleVoter},
		{"voter8", "voter123", types.RoleVoter},
		{"voter9", "voter123", types.RoleVoter},
		{"voter10", "voter123", types.RoleVoter},
	}

	now := time.Now().Format(constants.TIME_FORMAT)

	for _, u := range users {
		hashSalt, err := s.hasher.GenerateHash([]byte(u.password), []byte{})
		if err != nil {
			return err
		}

		user := types.User{
			Username: u.username,
			Password: string(hashSalt.Hash),
			Salt:     hex.EncodeToString(hashSalt.Salt),
			Role:     u.role,
			Date:     now,
			Version:  1,
		}

		conditions := bson.D{{Key: "username", Value: u.username}}
		_, _, err = s.db.InsertDocument(ctx, mongo.DbCollections[mongo.UserCollection], &conditions, &user)
		if err != nil {
			// Skip if user already exists
			logger.Debug("User may already exist", "username", u.username)
			continue
		}
		logger.Info("Created user", "username", u.username, "role", u.role)
	}

	return nil
}

// seedPolls creates sample polls in various states.
func (s *Seeder) seedPolls(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	// Get an official user to be the creator
	var officials []types.User
	_, err := s.db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "role", Value: types.RoleOfficial}},
		nil,
		&officials,
	)
	if err != nil || len(officials) == 0 {
		logger.Warn("No officials found, skipping poll seeding")
		return nil
	}

	creatorID := officials[0].ID
	now := time.Now()
	nowStr := now.Format(constants.TIME_FORMAT)

	polls := []types.Poll{
		{
			Title:       "Board Election 2024",
			Description: "Annual election for board members. Vote for your preferred candidates.",
			CreatorID:   creatorID,
			Options: []types.PollOption{
				{ID: "candidate_a", Label: "Alice Johnson"},
				{ID: "candidate_b", Label: "Bob Smith"},
				{ID: "candidate_c", Label: "Carol Williams"},
				{ID: "candidate_d", Label: "David Brown"},
			},
			Status:            types.PollStatusOpen,
			StartTime:         &now,
			AuditorThreshold:  1,
			AuditorTotal:      2,
			OfficialThreshold: 2,
			OfficialTotal:     3,
			CreatedAt:         nowStr,
			UpdatedAt:         nowStr,
			Version:           1,
		},
		{
			Title:       "Company Policy Update",
			Description: "Vote on the proposed changes to the remote work policy.",
			CreatorID:   creatorID,
			Options: []types.PollOption{
				{ID: "approve", Label: "Approve Changes"},
				{ID: "reject", Label: "Reject Changes"},
				{ID: "abstain", Label: "Abstain"},
			},
			Status:            types.PollStatusDraft,
			AuditorThreshold:  1,
			AuditorTotal:      2,
			OfficialThreshold: 2,
			OfficialTotal:     3,
			CreatedAt:         nowStr,
			UpdatedAt:         nowStr,
			Version:           1,
		},
		{
			Title:       "Budget Allocation Q1 2025",
			Description: "Vote on the proposed budget allocation for the first quarter.",
			CreatorID:   creatorID,
			Options: []types.PollOption{
				{ID: "option_a", Label: "Plan A - Focus on R&D"},
				{ID: "option_b", Label: "Plan B - Focus on Marketing"},
				{ID: "option_c", Label: "Plan C - Balanced Approach"},
			},
			Status:            types.PollStatusOpen,
			StartTime:         &now,
			AuditorThreshold:  1,
			AuditorTotal:      3,
			OfficialThreshold: 2,
			OfficialTotal:     4,
			CreatedAt:         nowStr,
			UpdatedAt:         nowStr,
			Version:           1,
		},
		{
			Title:       "Community Project Selection",
			Description: "Select the community project we will support this year.",
			CreatorID:   creatorID,
			Options: []types.PollOption{
				{ID: "project_1", Label: "Local Food Bank"},
				{ID: "project_2", Label: "Youth Education Program"},
				{ID: "project_3", Label: "Environmental Cleanup"},
				{ID: "project_4", Label: "Senior Care Initiative"},
			},
			Status:            types.PollStatusClosed,
			AuditorThreshold:  1,
			AuditorTotal:      2,
			OfficialThreshold: 2,
			OfficialTotal:     3,
			CreatedAt:         nowStr,
			UpdatedAt:         nowStr,
			Version:           1,
		},
	}

	for _, poll := range polls {
		_, _, err := s.db.InsertDocument(ctx, mongo.DbCollections[mongo.PollCollection], nil, &poll)
		if err != nil {
			logger.Error("Failed to create poll", "title", poll.Title, "error", err)
			continue
		}
		logger.Info("Created poll", "title", poll.Title, "status", poll.Status)
	}

	return nil
}
