package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/culbec/CRYPTO-sss/src/backend/pkg"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Create a valid config file
	validConfig := map[string]interface{}{
		"db_uri":         "mongodb://localhost:27017",
		"db_name":        "testdb",
		"jwt_secret_key": "test-secret-key",
		"server_host":    "127.0.0.1",
		"server_port":    "3000",
	}

	configJSON, err := json.Marshal(validConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test LoadConfig
	config, err := pkg.LoadConfig(&configFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify all fields
	if config.DbURI != "mongodb://localhost:27017" {
		t.Errorf("DbURI = %q, want %q", config.DbURI, "mongodb://localhost:27017")
	}
	if config.DbName != "testdb" {
		t.Errorf("DbName = %q, want %q", config.DbName, "testdb")
	}
	if config.JwtSecretKey != "test-secret-key" {
		t.Errorf("JwtSecretKey = %q, want %q", config.JwtSecretKey, "test-secret-key")
	}
	if config.ServerHost != "127.0.0.1" {
		t.Errorf("ServerHost = %q, want %q", config.ServerHost, "127.0.0.1")
	}
	if config.ServerPort != "3000" {
		t.Errorf("ServerPort = %q, want %q", config.ServerPort, "3000")
	}
	if config.ConfigPath != configFile {
		t.Errorf("ConfigPath = %q, want %q", config.ConfigPath, configFile)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Create invalid JSON file
	invalidJSON := []byte(`{"db_uri": "invalid json"`)
	if err := os.WriteFile(configFile, invalidJSON, 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	// Test LoadConfig with invalid JSON
	_, err := pkg.LoadConfig(&configFile)
	if err == nil {
		t.Error("LoadConfig() should fail with invalid JSON")
	}
}

func TestLoadConfig_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.json")

	// Test LoadConfig with non-existent file
	_, err := pkg.LoadConfig(&nonExistentFile)
	if err == nil {
		t.Error("LoadConfig() should fail when config file does not exist")
	}
}

func TestLoadConfig_NoConfigFileNil(t *testing.T) {
	// Test LoadConfig with nil (should try to use chooseConfigFile)
	// This will fail if no config files exist in the default locations
	_, err := pkg.LoadConfig(nil)
	if err == nil {
		t.Error("LoadConfig() should fail when no config file exists and nil is passed")
	}
}

func TestLoadConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	configJSON := []byte(`{
		"db_uri": "mongodb://localhost:27017/?maxPoolSize=20&w=majority",
		"db_name": "testdb",
		"jwt_secret_key": "test-secret-key-12345",
		"server_host": "0.0.0.0",
		"server_port": "8080"
	}`)

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test LoadConfig
	config, err := pkg.LoadConfig(&configFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify all fields
	if config.DbURI != "mongodb://localhost:27017/?maxPoolSize=20&w=majority" {
		t.Errorf("DbURI = %q, want %q", config.DbURI, "mongodb://localhost:27017/?maxPoolSize=20&w=majority")
	}
	if config.DbName != "testdb" {
		t.Errorf("DbName = %q, want %q", config.DbName, "testdb")
	}
	if config.JwtSecretKey != "test-secret-key-12345" {
		t.Errorf("JwtSecretKey = %q, want %q", config.JwtSecretKey, "test-secret-key-12345")
	}
	if config.ServerHost != "0.0.0.0" {
		t.Errorf("ServerHost = %q, want %q", config.ServerHost, "0.0.0.0")
	}
	if config.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", config.ServerPort, "8080")
	}
	if config.ConfigPath != configFile {
		t.Errorf("ConfigPath = %q, want %q", config.ConfigPath, configFile)
	}
}

func TestConfig_JSONTags(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Test that JSON tags are correctly defined
	configJSON := []byte(`{
		"db_uri": "mongodb://localhost:27017",
		"db_name": "testdb",
		"jwt_secret_key": "secret",
		"server_host": "localhost",
		"server_port": "3000"
	}`)

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test LoadConfig to verify JSON tags work
	config, err := pkg.LoadConfig(&configFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify all fields are populated
	if config.DbURI == "" {
		t.Error("DbURI should be populated from db_uri JSON field")
	}
	if config.DbName == "" {
		t.Error("DbName should be populated from db_name JSON field")
	}
	if config.JwtSecretKey == "" {
		t.Error("JwtSecretKey should be populated from jwt_secret_key JSON field")
	}
	if config.ServerHost == "" {
		t.Error("ServerHost should be populated from server_host JSON field")
	}
	if config.ServerPort == "" {
		t.Error("ServerPort should be populated from server_port JSON field")
	}
}

func TestConfig_MarshalUnmarshal(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	originalConfig := &pkg.Config{
		DbURI:        "mongodb://localhost:27017",
		DbName:       "testdb",
		JwtSecretKey: "test-secret",
		ServerHost:   "127.0.0.1",
		ServerPort:   "3000",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load using LoadConfig
	loadedConfig, err := pkg.LoadConfig(&configFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify all fields
	if loadedConfig.DbURI != originalConfig.DbURI {
		t.Errorf("DbURI = %q, want %q", loadedConfig.DbURI, originalConfig.DbURI)
	}
	if loadedConfig.DbName != originalConfig.DbName {
		t.Errorf("DbName = %q, want %q", loadedConfig.DbName, originalConfig.DbName)
	}
	if loadedConfig.JwtSecretKey != originalConfig.JwtSecretKey {
		t.Errorf("JwtSecretKey = %q, want %q", loadedConfig.JwtSecretKey, originalConfig.JwtSecretKey)
	}
	if loadedConfig.ServerHost != originalConfig.ServerHost {
		t.Errorf("ServerHost = %q, want %q", loadedConfig.ServerHost, originalConfig.ServerHost)
	}
	if loadedConfig.ServerPort != originalConfig.ServerPort {
		t.Errorf("ServerPort = %q, want %q", loadedConfig.ServerPort, originalConfig.ServerPort)
	}
	if loadedConfig.ConfigPath != configFile {
		t.Errorf("ConfigPath = %q, want %q", loadedConfig.ConfigPath, configFile)
	}
}
