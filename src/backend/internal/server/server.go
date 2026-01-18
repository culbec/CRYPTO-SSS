package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/seeder"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/gin-gonic/gin"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	config   *pkg.Config
	router   *gin.Engine
	dbClient *mongo.Client
	ctx      context.Context
}

// New creates a new Server instance with all dependencies initialized.
func New(ctx context.Context, config *pkg.Config) (*Server, error) {
	logger := logging.FromContext(ctx)

	logger.Info("Preparing the DB client...")
	dbClient, err := mongo.PrepareClient(ctx, config)
	if err != nil {
		logger.Error("Error preparing the DB client", "error", err)
		return nil, err
	}
	logger.Info("DB client prepared!")

	// Seed database if configured
	if config.SeedData {
		logger.Info("Seed data enabled, initializing seeder...")
		s := seeder.NewSeeder(dbClient)
		if err := s.SeedAll(ctx); err != nil {
			logger.Error("Error seeding database", "error", err)
			// Continue anyway - seeding failure should not prevent startup
		}
	}

	router := gin.Default()

	srv := &Server{
		config:   config,
		router:   router,
		dbClient: dbClient,
		ctx:      ctx,
	}

	srv.setupMiddleware()
	srv.setupRoutes()

	return srv, nil
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	logger := logging.FromContext(s.ctx)

	host := s.config.ServerHost
	port := s.config.ServerPort

	if host == "" {
		logger.Warn("Server host not set, using default", "default", internal.SERVER_HOST)
		host = internal.SERVER_HOST
	}
	if port == "" {
		logger.Warn("Server port not set, using default", "default", internal.SERVER_PORT)
		port = internal.SERVER_PORT
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	logger.Info("Starting server...", "address", addr)

	return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server and releases resources.
func (s *Server) Shutdown() error {
	logger := logging.FromContext(s.ctx)

	logger.Info("Shutting down server...")

	if s.dbClient != nil {
		if err := mongo.Cleanup(s.ctx, s.dbClient); err != nil {
			logger.Error("Error cleaning up the DB client", "error", err)
			return err
		}
	}

	logger.Info("Server shutdown complete")
	return nil
}

// Router returns the underlying gin.Engine for testing purposes.
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Health returns a simple health check handler.
func (s *Server) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
