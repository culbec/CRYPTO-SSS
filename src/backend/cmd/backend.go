package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/gin-gonic/gin"
)

// corsMiddleware: middleware to enable CORS.
// Allows all origins, credentials, headers, and methods.
// If the request method is OPTIONS, it will abort with a 204 status code.
// Returns the gin.HandlerFunc.
func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins temporarily
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization, Access-Control-Allow-Origin")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

func prepareHandlers(router *gin.Engine, ctx context.Context, config *pkg.Config, client *mongo.Client) {
	logger := logging.FromContext(ctx)

	secretKey := config.JwtSecretKey
	if secretKey == "" {
		logger.Error("JWT secret key not set, using default")
		panic("JWT secret key not set")
	}

	// mock ping-pong endpoint
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

}

func main() {
	logger := logging.InitLogger(internal.LOG_FILE)
	defer logging.CloseLogger()

	ctx := logging.WithContext(context.Background(), logger)

	logger.Info("App starting...")

	logger.Info("Loading config...")
	config, err := pkg.LoadConfig(nil)
	if err != nil {
		logger.Error("Error loading config", "error", err)
		panic(err)
	}
	logger.Info("Config loaded!")

	serverHost, serverPort := config.ServerHost, config.ServerPort
	if serverHost == "" {
		logger.Warn("Server host not set, using default", "default", internal.SERVER_HOST)
		serverHost = internal.SERVER_HOST
	}
	if serverPort == "" {
		logger.Warn("Server port not set, using default", "default", internal.SERVER_PORT)
		serverPort = internal.SERVER_PORT
	}

	logger.Info("Starting server...", "host", serverHost, "port", serverPort)

	router := gin.Default()

	logger.Info("Enabling default CORS configuration...")
	router.Use(corsMiddleware())

	logger.Info("Preparing the DB client...")
	client, err := mongo.PrepareClient(ctx, config)
	if err != nil {
		logger.Error("Error preparing the DB client", "error", err)
	} else {
		logger.Info("DB client prepared!")
		defer func() {
			err := mongo.Cleanup(ctx, client)
			if err != nil {
				logger.Error("Error cleaning up the DB client", "error", err)
			}
		}()
	}

	logger.Info("Preparing the handlers...")
	prepareHandlers(router, ctx, config, client)
	logger.Info("Handlers prepared!")

	server := fmt.Sprintf("%s:%s", serverHost, serverPort)
	err = router.Run(server)
	if err != nil {
		logger.Error("Error starting the server", "error", err)
		panic(err)
	}

	logger.Info("Server started!", "server", server)

}
