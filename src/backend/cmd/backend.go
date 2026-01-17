// Package main is the entry point for the backend application.
//
//	@title						CRYPTO-SSS Backend API
//	@version					1.0
//	@description				Backend API for the CRYPTO-SSS project providing authentication and secret sharing services.
//	@termsOfService				http://swagger.io/terms/
//	@contact.name				API Support
//	@contact.email				support@crypto-sss.io
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//	@host						localhost:3000
//	@BasePath					/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter JWT token directly (Bearer prefix is optional).
package main

import (
	"context"

	"github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/server"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg"
	"github.com/gin-gonic/gin"
)

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
	logger.Info("Config loaded!", "path", config.ConfigPath)

	// Setting release mode if specified in the config
	if config.ReleaseMode {
		logger.Info("Setting release mode...")
		gin.SetMode(gin.ReleaseMode)
	} else {
		logger.Info("Setting debug mode...")
		gin.SetMode(gin.DebugMode)
	}

	srv, err := server.New(ctx, config)
	if err != nil {
		logger.Error("Error creating server", "error", err)
		panic(err)
	}
	defer func() {
		if err := srv.Shutdown(); err != nil {
			logger.Error("Error during shutdown", "error", err)
		}
	}()

	if err := srv.Run(); err != nil {
		logger.Error("Error running server", "error", err)
		panic(err)
	}
}
