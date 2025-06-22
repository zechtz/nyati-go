package appconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zechtz/nyatictl/logger"
)

// Config represents the application configuration
type Config struct {
	// Web server configuration
	WebMode bool   `env:"NYATI_WEB_MODE" default:"false"`
	Port    string `env:"NYATI_PORT" default:"8080"`
	
	// Database configuration
	DatabasePath       string        `env:"NYATI_DB_PATH" default:"./nyatictl.db"`
	DatabaseMaxConns   int           `env:"NYATI_DB_MAX_CONNS" default:"25"`
	DatabaseIdleConns  int           `env:"NYATI_DB_IDLE_CONNS" default:"5"`
	DatabaseConnLife   time.Duration `env:"NYATI_DB_CONN_LIFETIME" default:"300s"`
	DatabaseIdleTime   time.Duration `env:"NYATI_DB_IDLE_TIME" default:"60s"`
	
	// Logging configuration
	LogPath           string       `env:"NYATI_LOG_PATH" default:"nyatictl.log"`
	LogLevel          string       `env:"NYATI_LOG_LEVEL" default:"INFO"`
	StructuredLogging bool         `env:"NYATI_STRUCTURED_LOGGING" default:"false"`
	
	// File paths
	ConfigsPath string `env:"NYATI_CONFIGS_PATH" default:"configs.json"`
	
	// Security settings
	JWTSecret        string        `env:"NYATI_JWT_SECRET" default:""`
	SessionTimeout   time.Duration `env:"NYATI_SESSION_TIMEOUT" default:"24h"`
	
	// Performance settings
	RequestTimeout   time.Duration `env:"NYATI_REQUEST_TIMEOUT" default:"30s"`
	ShutdownTimeout  time.Duration `env:"NYATI_SHUTDOWN_TIMEOUT" default:"10s"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{}
	
	// Load each field using reflection-like approach
	if err := loadField(cfg, "WebMode", "NYATI_WEB_MODE", "false"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "Port", "NYATI_PORT", "8080"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "DatabasePath", "NYATI_DB_PATH", "./nyatictl.db"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "DatabaseMaxConns", "NYATI_DB_MAX_CONNS", "25"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "DatabaseIdleConns", "NYATI_DB_IDLE_CONNS", "5"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "DatabaseConnLife", "NYATI_DB_CONN_LIFETIME", "300s"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "DatabaseIdleTime", "NYATI_DB_IDLE_TIME", "60s"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "LogPath", "NYATI_LOG_PATH", "nyatictl.log"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "LogLevel", "NYATI_LOG_LEVEL", "INFO"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "StructuredLogging", "NYATI_STRUCTURED_LOGGING", "false"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "ConfigsPath", "NYATI_CONFIGS_PATH", "configs.json"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "JWTSecret", "NYATI_JWT_SECRET", ""); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "SessionTimeout", "NYATI_SESSION_TIMEOUT", "24h"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "RequestTimeout", "NYATI_REQUEST_TIMEOUT", "30s"); err != nil {
		return nil, err
	}
	if err := loadField(cfg, "ShutdownTimeout", "NYATI_SHUTDOWN_TIMEOUT", "10s"); err != nil {
		return nil, err
	}
	
	return cfg, nil
}

// loadField loads a configuration field from environment variable or uses default
func loadField(cfg *Config, fieldName, envName, defaultValue string) error {
	value := getEnvOrDefault(envName, defaultValue)
	
	switch fieldName {
	case "WebMode":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %v", envName, err)
		}
		cfg.WebMode = parsed
	case "Port":
		cfg.Port = value
	case "DatabasePath":
		cfg.DatabasePath = value
	case "DatabaseMaxConns":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %v", envName, err)
		}
		cfg.DatabaseMaxConns = parsed
	case "DatabaseIdleConns":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %v", envName, err)
		}
		cfg.DatabaseIdleConns = parsed
	case "DatabaseConnLife":
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %v", envName, err)
		}
		cfg.DatabaseConnLife = parsed
	case "DatabaseIdleTime":
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %v", envName, err)
		}
		cfg.DatabaseIdleTime = parsed
	case "LogPath":
		cfg.LogPath = value
	case "LogLevel":
		cfg.LogLevel = strings.ToUpper(value)
	case "StructuredLogging":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %v", envName, err)
		}
		cfg.StructuredLogging = parsed
	case "ConfigsPath":
		cfg.ConfigsPath = value
	case "JWTSecret":
		cfg.JWTSecret = value
	case "SessionTimeout":
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %v", envName, err)
		}
		cfg.SessionTimeout = parsed
	case "RequestTimeout":
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %v", envName, err)
		}
		cfg.RequestTimeout = parsed
	case "ShutdownTimeout":
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration value for %s: %v", envName, err)
		}
		cfg.ShutdownTimeout = parsed
	default:
		return fmt.Errorf("unknown field: %s", fieldName)
	}
	
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return defaultValue
}

// Validate validates the configuration values
func (cfg *Config) Validate() error {
	// Validate port
	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %s (must be between 1 and 65535)", cfg.Port)
	}
	
	// Validate database connections
	if cfg.DatabaseMaxConns < 1 {
		return fmt.Errorf("database max connections must be at least 1, got %d", cfg.DatabaseMaxConns)
	}
	if cfg.DatabaseIdleConns < 0 {
		return fmt.Errorf("database idle connections cannot be negative, got %d", cfg.DatabaseIdleConns)
	}
	if cfg.DatabaseIdleConns > cfg.DatabaseMaxConns {
		return fmt.Errorf("database idle connections (%d) cannot exceed max connections (%d)", 
			cfg.DatabaseIdleConns, cfg.DatabaseMaxConns)
	}
	
	// Validate durations
	if cfg.DatabaseConnLife < time.Second {
		return fmt.Errorf("database connection lifetime must be at least 1 second, got %v", cfg.DatabaseConnLife)
	}
	if cfg.DatabaseIdleTime < 0 {
		return fmt.Errorf("database idle time cannot be negative, got %v", cfg.DatabaseIdleTime)
	}
	if cfg.SessionTimeout < time.Minute {
		return fmt.Errorf("session timeout must be at least 1 minute, got %v", cfg.SessionTimeout)
	}
	if cfg.RequestTimeout < time.Second {
		return fmt.Errorf("request timeout must be at least 1 second, got %v", cfg.RequestTimeout)
	}
	if cfg.ShutdownTimeout < time.Second {
		return fmt.Errorf("shutdown timeout must be at least 1 second, got %v", cfg.ShutdownTimeout)
	}
	
	// Validate log level
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}
	if !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be one of: DEBUG, INFO, WARN, ERROR, FATAL)", cfg.LogLevel)
	}
	
	// Validate paths are not empty
	if cfg.LogPath == "" {
		return fmt.Errorf("log path cannot be empty")
	}
	if cfg.ConfigsPath == "" {
		return fmt.Errorf("configs path cannot be empty")
	}
	if cfg.DatabasePath == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	
	// Warn if JWT secret is not set (but don't fail validation)
	if cfg.JWTSecret == "" {
		logger.Warn("JWT secret not configured - using default (SECURITY RISK in production)")
	}
	
	return nil
}

// GetLogLevel returns the logger.LogLevel corresponding to the configured log level
func (cfg *Config) GetLogLevel() logger.LogLevel {
	switch cfg.LogLevel {
	case "DEBUG":
		return logger.DEBUG
	case "INFO":
		return logger.INFO
	case "WARN":
		return logger.WARN
	case "ERROR":
		return logger.ERROR
	case "FATAL":
		return logger.FATAL
	default:
		return logger.INFO
	}
}

// GetDatabaseURL constructs the SQLite database connection URL with parameters
func (cfg *Config) GetDatabaseURL() string {
	return fmt.Sprintf("%s?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=1",
		cfg.DatabasePath)
}

// LogConfiguration logs the current configuration (excluding sensitive values)
func (cfg *Config) LogConfiguration() {
	logger.Info("Application configuration loaded", map[string]interface{}{
		"web_mode":            cfg.WebMode,
		"port":                cfg.Port,
		"database_path":       cfg.DatabasePath,
		"database_max_conns":  cfg.DatabaseMaxConns,
		"database_idle_conns": cfg.DatabaseIdleConns,
		"log_path":            cfg.LogPath,
		"log_level":           cfg.LogLevel,
		"structured_logging":  cfg.StructuredLogging,
		"configs_path":        cfg.ConfigsPath,
		"jwt_secret_set":      cfg.JWTSecret != "",
		"session_timeout":     cfg.SessionTimeout.String(),
		"request_timeout":     cfg.RequestTimeout.String(),
		"shutdown_timeout":    cfg.ShutdownTimeout.String(),
	})
}