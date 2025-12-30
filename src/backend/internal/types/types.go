package types

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ObjectId = primitive.ObjectID

// User struct
type User struct {
	ID ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
	Salt string `json:"salt" bson:"salt"`
	Date string `json:"date" bson:"date"`
	Version int `json:"version" bson:"version"`
}

// LoginRequest struct
type LoginRequest struct {
	Username string `json:"username" bson:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest struct
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse struct
type AuthResponse struct {
	UserID string `json:"user_id"`
	Token string `json:"token"`
}