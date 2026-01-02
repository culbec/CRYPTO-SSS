package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const ContextUsernameKey = "username"

// RequireAuth validates the Authorization bearer token and stores the username in the Gin context.
func RequireAuth(auth *AuthHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		username, err := auth.ValidateToken(ctx)
		if err != nil {
			// ValidateToken already wrote the response.
			return
		}

		ctx.Set(ContextUsernameKey, username)
		ctx.Next()
	}
}

// Optional helper for handlers that need the authenticated username.
func UsernameFromContext(ctx *gin.Context) (string, bool) {
	v, ok := ctx.Get(ContextUsernameKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// RequireNoAuth is a small convenience middleware that rejects requests that already have a valid token.
// Useful for routes like /register or /login if desired.
func RequireNoAuth(auth *AuthHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, err := auth.ValidateToken(ctx)
		if err == nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "already authenticated"})
			return
		}
		ctx.Next()
	}
}
