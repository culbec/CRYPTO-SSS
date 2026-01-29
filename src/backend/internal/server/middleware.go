package server

import (
	"log/slog"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/gin-gonic/gin"
)

// setupMiddleware configures all middleware for the server.
func (s *Server) setupMiddleware() {
	logger := logging.FromContext(s.ctx)

	s.router.Use(loggerMiddleware(logger))

	logger.Info("Enabling CORS configuration...")
	s.router.Use(corsMiddleware())
}

// loggerMiddleware injects the logger into the request context.
func loggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqCtx := logging.WithContext(ctx.Request.Context(), logger)
		ctx.Request = ctx.Request.WithContext(reqCtx)
		ctx.Next()
	}
}

// corsMiddleware enables CORS for all origins.
// Allows all origins, credentials, headers, and methods.
// If the request method is OPTIONS, it aborts with a 204 status code.
func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Requested-With, Origin")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		ctx.Writer.Header().Set("Access-Control-Max-Age", "86400")
		ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}
