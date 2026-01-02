package test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		logPath string
	}{
		{
			name:    "initializes logger successfully",
			logPath: "../logs/test.log",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logging.InitLogger(tt.logPath)
			if got == nil {
				t.Errorf("InitLogger() = nil, want non-nil logger")
			}
			// Cleanup
			logging.CloseLogger()
		})
	}
}

func TestWithContext(t *testing.T) {
	tests := []struct {
		name   string // description of this test case
		ctx    context.Context
		logger *slog.Logger
	}{
		{
			name:   "attaches logger to background context",
			ctx:    context.Background(),
			logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
		},
		{
			name:   "attaches logger to context with existing values",
			ctx:    context.WithValue(context.Background(), "key", "value"),
			logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logging.WithContext(tt.ctx, tt.logger)
			if got == nil {
				t.Errorf("WithContext() = nil, want non-nil context")
			}
			// Verify logger is in context
			retrievedLogger := logging.FromContext(got)
			if retrievedLogger != tt.logger {
				t.Errorf("WithContext() logger mismatch, got %v, want %v", retrievedLogger, tt.logger)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		ctx  context.Context
	}{
		{
			name: "returns logger from context",
			ctx:  logging.WithContext(context.Background(), slog.New(slog.NewTextHandler(os.Stdout, nil))),
		},
		{
			name: "returns default logger when context has no logger",
			ctx:  context.Background(),
		},
		{
			name: "returns default logger when context has other values",
			ctx:  context.WithValue(context.Background(), "key", "value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logging.FromContext(tt.ctx)
			if got == nil {
				t.Errorf("FromContext() = nil, want non-nil logger")
			}
		})
	}
}

func TestGetDefaultLogger(t *testing.T) {
	tests := []struct {
		name string // description of this test case
	}{
		{
			name: "returns default logger when initialized",
		},
		{
			name: "initializes logger if not already initialized",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logging.GetDefaultLogger()
			if got == nil {
				t.Errorf("GetDefaultLogger() = nil, want non-nil logger")
			}
		})
	}
}

func TestCloseLogger(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		logPath string
	}{
		{
			name:    "closes log file successfully",
			logPath: "../logs/test.log",
		},
		{
			name:    "handles multiple close calls gracefully",
			logPath: "../logs/test.log",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize logger first to create log file
			logging.InitLogger(tt.logPath)
			// Close should not panic
			logging.CloseLogger()
			// Second close should also not panic
			logging.CloseLogger()
		})
	}
}
