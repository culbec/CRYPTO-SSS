// Package helpers provides shared utility functions for API handlers.
package helpers

import (
	"context"
	"errors"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	// ErrUserNotFound is returned when a user lookup fails.
	ErrUserNotFound = errors.New("user not found")
	// ErrPollNotFound is returned when a poll lookup fails.
	ErrPollNotFound = errors.New("poll not found")
)

// GetUserByUsername retrieves a user by their username.
func GetUserByUsername(ctx context.Context, db *mongo.Client, username string) (*types.User, error) {
	var users []types.User
	_, err := db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "username", Value: username}},
		nil,
		&users,
	)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return &users[0], nil
}

// GetUserByID retrieves a user by their ID.
func GetUserByID(ctx context.Context, db *mongo.Client, userID primitive.ObjectID) (*types.User, error) {
	var users []types.User
	_, err := db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "_id", Value: userID}},
		nil,
		&users,
	)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return &users[0], nil
}

// GetUsersByRole retrieves users by their role, limited to the specified count.
// If limit <= 0 or limit exceeds available users, all matching users are returned.
func GetUsersByRole(ctx context.Context, db *mongo.Client, role types.UserRole, limit int) ([]types.User, error) {
	var users []types.User
	_, err := db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "role", Value: role}},
		nil,
		&users,
	)
	if err != nil {
		return nil, err
	}
	// If limit <= 0 or limit exceeds available users, return all.
	if limit <= 0 || len(users) <= limit {
		return users, nil
	}
	return users[:limit], nil
}

// GetPollByID retrieves a poll by its ID.
func GetPollByID(ctx context.Context, db *mongo.Client, pollID primitive.ObjectID) (*types.Poll, error) {
	var polls []types.Poll
	_, err := db.QueryCollection(
		ctx,
		mongo.DbCollections[mongo.PollCollection],
		&bson.D{{Key: "_id", Value: pollID}},
		nil,
		&polls,
	)
	if err != nil {
		return nil, err
	}
	if len(polls) == 0 {
		return nil, ErrPollNotFound
	}
	return &polls[0], nil
}

// ValidateUserRole checks if the given role string is valid.
func ValidateUserRole(role types.UserRole) bool {
	switch role {
	case types.RoleVoter, types.RoleAuditor, types.RoleOfficial, types.RoleAdmin:
		return true
	default:
		return false
	}
}
