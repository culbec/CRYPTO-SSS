package logging

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	slogmulti "github.com/samber/slog-multi"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
)

// loggerKey: key for storing the logger in the context.
type loggerKey struct{}

var (
	logFile       *os.File
	defaultLogger *slog.Logger
	mutex         sync.RWMutex
)

// InitLogger: initializes the logger, with both console and file logging.
func InitLogger(logPath string) *slog.Logger {
	mutex.Lock()
	defer mutex.Unlock()

	// Check if already initialized to prevent race conditions
	if defaultLogger != nil {
		return defaultLogger
	}

	// Console handler
	// Default if file creation is not possible
	consoleHandler := slog.NewTextHandler(os.Stdout, nil)
	fanout := slogmulti.Fanout(consoleHandler)
	defaultLogger = slog.New(fanout)

	// File handler
	var fileHandler slog.Handler = nil
	logDir := filepath.Dir(logPath)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
		return defaultLogger
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return defaultLogger
	}

	logFile = file
	fileHandler = slog.NewTextHandler(file, nil)

	// Signal channel to close the logger after process termination
	// Closes the logger and exits from the goroutine on crash
	// User must close the logger manually on each usage
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		CloseLogger()
		os.Exit(0)
	}()

	// Console + File handlers
	fanout = slogmulti.Fanout(consoleHandler, fileHandler)
	defaultLogger = slog.New(fanout)
	return defaultLogger
}

// WithContext: returns a new context with the logger attached.
// Used to pass the logger to the context for further use.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext: returns the logger from the context.
// Used to get the logger from the context for further use.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return GetDefaultLogger() // default logger if not found
}

// GetDefaultLogger: returns the default logger.
func GetDefaultLogger() *slog.Logger {
	mutex.RLock()
	if defaultLogger != nil {
		defer mutex.RUnlock()
		return defaultLogger
	}
	mutex.RUnlock()

	return InitLogger(constants.LOG_FILE)
}

// CloseLogger: closes the log file and resets the log file pointer.
func CloseLogger() {
	mutex.Lock()
	defer mutex.Unlock()

	if logFile != nil {
		if err := logFile.Close(); err != nil {
			log.Printf("Failed to close log file: %v", err)
		}
		logFile = nil
	}
}
