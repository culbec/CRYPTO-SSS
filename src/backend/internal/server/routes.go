package server

import (
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/docs"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/admin"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/auth"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/ballot"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/poll"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes configures all API routes for the server.
func (s *Server) setupRoutes() {
	logger := logging.FromContext(s.ctx)
	logger.Info("Preparing the handlers...")

	secretKey := s.config.JwtSecretKey
	if secretKey == "" {
		logger.Error("JWT secret key not set")
		panic("JWT secret key not set")
	}

	// Swagger documentation
	s.setupSwagger()

	// Health check endpoint
	s.router.GET("/api/health", s.Health)

	// Ping-pong endpoint for testing
	s.router.GET("/api/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// Auth handlers
	authHandler := auth.NewAuthHandler(s.dbClient, []byte(secretKey))
	s.registerAuthRoutes(authHandler)

	// Poll handlers
	pollHandler := poll.NewPollHandler(s.dbClient, s)
	s.registerPollRoutes(pollHandler, authHandler)

	// Ballot handlers
	ballotHandler := ballot.NewBallotHandler(s.dbClient, s)
	s.registerBallotRoutes(ballotHandler, authHandler)

	// Admin handlers
	adminHandler := admin.NewAdminHandler(s.dbClient)
	s.registerAdminRoutes(adminHandler, authHandler)

	logger.Info("Handlers prepared!")
}

// setupSwagger configures Swagger documentation routes.
func (s *Server) setupSwagger() {
	// Update Swagger info with actual host
	host := s.config.ServerHost
	port := s.config.ServerPort
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "3000"
	}
	
	// For 0.0.0.0, use localhost in Swagger docs
	if host == "0.0.0.0" {
		host = "localhost"
	}
	
	docs.SwaggerInfo.Host = host + ":" + port
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// registerAuthRoutes registers all authentication-related routes.
func (s *Server) registerAuthRoutes(authHandler *auth.AuthHandler) {
	// Public auth routes
	publicAuth := s.router.Group("/api/auth")
	{
		publicAuth.POST("/login", jsonContentType(), authHandler.LoginHandler)
		publicAuth.POST("/register", jsonContentType(), authHandler.RegisterHandler)
	}

	// Protected auth routes (require valid token)
	protectedAuth := s.router.Group("/api/auth")
	protectedAuth.Use(auth.RequireAuth(authHandler))
	{
		protectedAuth.POST("/logout", jsonContentType(), authHandler.LogoutHandler)
		protectedAuth.POST("/validate", jsonContentType(), authHandler.ValidateTokenHandler)
	}
}

// registerPollRoutes registers all poll-related routes.
func (s *Server) registerPollRoutes(pollHandler *poll.PollHandler, authHandler *auth.AuthHandler) {
	// All poll routes require authentication
	polls := s.router.Group("/api/polls")
	polls.Use(auth.RequireAuth(authHandler))
	{
		// List polls (all authenticated users)
		polls.GET("", jsonContentType(), pollHandler.ListPollsHandler)

		// Get single poll (all authenticated users)
		polls.GET("/:id", jsonContentType(), pollHandler.GetPollHandler)

		// Create poll (officials and admins only)
		polls.POST("", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleOfficial, types.RoleAdmin), pollHandler.CreatePollHandler)

		// Update poll status (officials and admins only)
		polls.PUT("/:id/status", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleOfficial, types.RoleAdmin), pollHandler.UpdatePollStatusHandler)

		// Freeze poll and distribute shares (officials and admins only)
		polls.POST("/:id/freeze", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleOfficial, types.RoleAdmin), pollHandler.FreezePollHandler)

		// Get my share (auditors and officials only)
		polls.GET("/:id/my-share", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleAuditor, types.RoleOfficial, types.RoleAdmin), pollHandler.GetMyShareHandler)

		// Get share status (all authenticated users)
		polls.GET("/:id/share-status", jsonContentType(), pollHandler.GetShareStatusHandler)

		// Reveal results (auditors and officials only)
		polls.POST("/:id/reveal", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleAuditor, types.RoleOfficial, types.RoleAdmin), pollHandler.RevealResultsHandler)

		// Get poll results (all authenticated users)
		polls.GET("/:id/results", jsonContentType(), pollHandler.GetPollResultsHandler)
	}
}

// registerBallotRoutes registers all ballot-related routes.
func (s *Server) registerBallotRoutes(ballotHandler *ballot.BallotHandler, authHandler *auth.AuthHandler) {
	// All ballot routes require authentication
	ballots := s.router.Group("/api/ballots")
	ballots.Use(auth.RequireAuth(authHandler))
	{
		// Cast ballot (voters only)
		ballots.POST("", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleVoter, types.RoleAdmin), ballotHandler.CastBallotHandler)

		// Get my ballot for a poll
		ballots.GET("/poll/:poll_id/my-ballot", jsonContentType(), ballotHandler.GetMyBallotHandler)

		// Contribute share for reveal (auditors and officials only)
		ballots.POST("/contribute-share", jsonContentType(), auth.RequireRole(s.dbClient, types.RoleAuditor, types.RoleOfficial, types.RoleAdmin), ballotHandler.ContributeShareHandler)
	}
}

// registerAdminRoutes registers all admin-only routes.
func (s *Server) registerAdminRoutes(adminHandler *admin.AdminHandler, authHandler *auth.AuthHandler) {
	// All admin routes require authentication and admin role
	adminGroup := s.router.Group("/api/admin")
	adminGroup.Use(auth.RequireAuth(authHandler))
	adminGroup.Use(auth.RequireRole(s.dbClient, types.RoleAdmin))
	{
		// SSS healthcheck - test secret sharing algorithm
		adminGroup.POST("/sss-healthcheck", jsonContentType(), adminHandler.SSSHealthCheckHandler)

		// SSS access structure test
		adminGroup.GET("/sss-access-test", jsonContentType(), adminHandler.SSSAccessStructureTestHandler)

		// User management
		adminGroup.GET("/users", jsonContentType(), adminHandler.ListUsersHandler)
		adminGroup.PUT("/users/:id/role", jsonContentType(), adminHandler.UpdateUserRoleHandler)
	}
}

// jsonContentType sets the Content-Type header to application/json.
func jsonContentType() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		ctx.Next()
	}
}
