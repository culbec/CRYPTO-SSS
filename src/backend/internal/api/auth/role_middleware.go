package auth

import (
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const ContextUserKey = "user"
const ContextUserRoleKey = "user_role"

// RequireRole middleware checks if the authenticated user has one of the required roles.
func RequireRole(db *mongo.Client, roles ...types.UserRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		username, ok := UsernameFromContext(ctx)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
			ctx.Abort()
			return
		}

		// Get user from database
		var users []types.User
		_, err := db.QueryCollection(
			ctx.Request.Context(),
			mongo.DbCollections[mongo.UserCollection],
			&bson.D{{Key: "username", Value: username}},
			nil,
			&users,
		)
		if err != nil || len(users) == 0 {
			ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "user not found"})
			ctx.Abort()
			return
		}

		user := &users[0]

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			ctx.JSON(http.StatusForbidden, types.ErrorResponse{Error: "insufficient permissions"})
			ctx.Abort()
			return
		}

		// Store user in context for handlers
		ctx.Set(ContextUserKey, user)
		ctx.Set(ContextUserRoleKey, user.Role)
		ctx.Next()
	}
}

// UserFromContext retrieves the authenticated user from the Gin context.
func UserFromContext(ctx *gin.Context) (*types.User, bool) {
	v, ok := ctx.Get(ContextUserKey)
	if !ok {
		return nil, false
	}
	user, ok := v.(*types.User)
	return user, ok
}

// RoleFromContext retrieves the authenticated user's role from the Gin context.
func RoleFromContext(ctx *gin.Context) (types.UserRole, bool) {
	v, ok := ctx.Get(ContextUserRoleKey)
	if !ok {
		return "", false
	}
	role, ok := v.(types.UserRole)
	return role, ok
}
