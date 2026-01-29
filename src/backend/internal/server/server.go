package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/api/auth"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/seeder"
	ws "github.com/culbec/CRYPTO-sss/src/backend/internal/websocket"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Server holds all dependencies for the HTTP server.
type Server struct {
	config   *pkg.Config
	router   *gin.Engine
	dbClient *mongo.Client
	ctx      context.Context
	hub      *ws.Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins
		return true
	},
}

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
		}
	}

	router := gin.Default()

	// Create WebSocket hub
	hub := ws.NewHub(logger)
	go hub.Run()

	srv := &Server{
		config:   config,
		router:   router,
		dbClient: dbClient,
		ctx:      ctx,
		hub:      hub,
	}

	// Setup middleware
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

	if s.hub != nil {
		s.hub.Shutdown()
	}

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

// registerWebSocket registers the /ws endpoint with auth middleware.
func (s *Server) registerWebSocket(authHandler *auth.AuthHandler) {
	s.router.GET("/ws", auth.RequireAuth(authHandler), s.handleWebSocket)
}

// handleWebSocket upgrades the HTTP connection to WebSocket and registers the client with the hub.
func (s *Server) handleWebSocket(c *gin.Context) {
	logger := logging.FromContext(s.ctx)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	clientID := uuid.New().String()
	client := &ws.Client{
		Hub:  s.hub,
		Conn: conn,
		Send: make(chan []byte, 256),
		ID:   clientID,
	}

	s.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// EmitEvent broadcasts an event to all connected WebSocket clients.
func (s *Server) EmitEvent(eventName string, data interface{}) {
	logger := logging.FromContext(s.ctx)
	if s.hub != nil {
		logger.Info("Emitting WebSocket event", "event", eventName, "clients", s.hub.ClientCount())
		s.hub.Emit(eventName, data)
	} else {
		logger.Warn("WebSocket hub not initialized", "event", eventName)
	}
}
