package appconfig

import (
	"os"
	"testing"
	"time"

	"github.com/zechtz/nyatictl/logger"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"NYATI_WEB_MODE",
		"NYATI_PORT",
		"NYATI_DB_PATH",
		"NYATI_DB_MAX_CONNS",
		"NYATI_DB_IDLE_CONNS",
		"NYATI_LOG_LEVEL",
		"NYATI_STRUCTURED_LOGGING",
	}
	
	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	
	// Restore environment after test
	defer func() {
		for _, envVar := range envVars {
			if value, exists := originalEnv[envVar]; exists && value != "" {
				os.Setenv(envVar, value)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	// Test default values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults
	if cfg.WebMode != false {
		t.Errorf("WebMode = %v, want false", cfg.WebMode)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}
	if cfg.DatabasePath != "./nyatictl.db" {
		t.Errorf("DatabasePath = %v, want ./nyatictl.db", cfg.DatabasePath)
	}
	if cfg.DatabaseMaxConns != 25 {
		t.Errorf("DatabaseMaxConns = %v, want 25", cfg.DatabaseMaxConns)
	}
	if cfg.DatabaseIdleConns != 5 {
		t.Errorf("DatabaseIdleConns = %v, want 5", cfg.DatabaseIdleConns)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel = %v, want INFO", cfg.LogLevel)
	}
	if cfg.StructuredLogging != false {
		t.Errorf("StructuredLogging = %v, want false", cfg.StructuredLogging)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"NYATI_WEB_MODE",
		"NYATI_PORT",
		"NYATI_DB_MAX_CONNS",
		"NYATI_LOG_LEVEL",
		"NYATI_STRUCTURED_LOGGING",
	}
	
	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
	}
	
	// Restore environment after test
	defer func() {
		for _, envVar := range envVars {
			if value, exists := originalEnv[envVar]; exists && value != "" {
				os.Setenv(envVar, value)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("NYATI_WEB_MODE", "true")
	os.Setenv("NYATI_PORT", "3000")
	os.Setenv("NYATI_DB_MAX_CONNS", "50")
	os.Setenv("NYATI_LOG_LEVEL", "DEBUG")
	os.Setenv("NYATI_STRUCTURED_LOGGING", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment variables are used
	if cfg.WebMode != true {
		t.Errorf("WebMode = %v, want true", cfg.WebMode)
	}
	if cfg.Port != "3000" {
		t.Errorf("Port = %v, want 3000", cfg.Port)
	}
	if cfg.DatabaseMaxConns != 50 {
		t.Errorf("DatabaseMaxConns = %v, want 50", cfg.DatabaseMaxConns)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("LogLevel = %v, want DEBUG", cfg.LogLevel)
	}
	if cfg.StructuredLogging != true {
		t.Errorf("StructuredLogging = %v, want true", cfg.StructuredLogging)
	}
}

func TestLoadWithInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid boolean for WebMode", "NYATI_WEB_MODE", "invalid"},
		{"invalid integer for DatabaseMaxConns", "NYATI_DB_MAX_CONNS", "invalid"},
		{"invalid duration for SessionTimeout", "NYATI_SESSION_TIMEOUT", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.envVar)
			defer func() {
				if original != "" {
					os.Setenv(tt.envVar, original)
				} else {
					os.Unsetenv(tt.envVar)
				}
			}()

			// Set invalid value
			os.Setenv(tt.envVar, tt.value)

			_, err := Load()
			if err == nil {
				t.Error("Load() should return error for invalid value")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Port:              "8080",
				DatabaseMaxConns:  25,
				DatabaseIdleConns: 5,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INFO",
				LogPath:           "test.log",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: &Config{
				Port:              "invalid",
				DatabaseMaxConns:  25,
				DatabaseIdleConns: 5,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INFO",
				LogPath:           "test.log",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: true,
		},
		{
			name: "invalid database max connections",
			cfg: &Config{
				Port:              "8080",
				DatabaseMaxConns:  0,
				DatabaseIdleConns: 5,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INFO",
				LogPath:           "test.log",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: true,
		},
		{
			name: "idle connections greater than max connections",
			cfg: &Config{
				Port:              "8080",
				DatabaseMaxConns:  5,
				DatabaseIdleConns: 10,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INFO",
				LogPath:           "test.log",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &Config{
				Port:              "8080",
				DatabaseMaxConns:  25,
				DatabaseIdleConns: 5,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INVALID",
				LogPath:           "test.log",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: true,
		},
		{
			name: "empty log path",
			cfg: &Config{
				Port:              "8080",
				DatabaseMaxConns:  25,
				DatabaseIdleConns: 5,
				DatabaseConnLife:  5 * time.Minute,
				DatabaseIdleTime:  1 * time.Minute,
				SessionTimeout:    24 * time.Hour,
				RequestTimeout:    30 * time.Second,
				ShutdownTimeout:   10 * time.Second,
				LogLevel:          "INFO",
				LogPath:           "",
				ConfigsPath:       "configs.json",
				DatabasePath:      "test.db",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		configLevel string
		expected    logger.LogLevel
	}{
		{"DEBUG", logger.DEBUG},
		{"INFO", logger.INFO},
		{"WARN", logger.WARN},
		{"ERROR", logger.ERROR},
		{"FATAL", logger.FATAL},
		{"INVALID", logger.INFO}, // fallback to INFO
	}

	for _, tt := range tests {
		t.Run(tt.configLevel, func(t *testing.T) {
			cfg := &Config{LogLevel: tt.configLevel}
			if got := cfg.GetLogLevel(); got != tt.expected {
				t.Errorf("Config.GetLogLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetDatabaseURL(t *testing.T) {
	cfg := &Config{DatabasePath: "/path/to/db.sqlite"}
	expected := "/path/to/db.sqlite?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=1"
	
	if got := cfg.GetDatabaseURL(); got != expected {
		t.Errorf("Config.GetDatabaseURL() = %v, want %v", got, expected)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save original value
	original := os.Getenv("TEST_ENV_VAR")
	defer func() {
		if original != "" {
			os.Setenv("TEST_ENV_VAR", original)
		} else {
			os.Unsetenv("TEST_ENV_VAR")
		}
	}()

	// Test with environment variable set
	os.Setenv("TEST_ENV_VAR", "env_value")
	if got := getEnvOrDefault("TEST_ENV_VAR", "default_value"); got != "env_value" {
		t.Errorf("getEnvOrDefault() = %v, want env_value", got)
	}

	// Test with environment variable not set
	os.Unsetenv("TEST_ENV_VAR")
	if got := getEnvOrDefault("TEST_ENV_VAR", "default_value"); got != "default_value" {
		t.Errorf("getEnvOrDefault() = %v, want default_value", got)
	}
}