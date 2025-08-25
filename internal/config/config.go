package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	CSVFile string
}

type LoggerConfig struct {
	Level  string
	Format string
}

type SecurityConfig struct {
	EnableCSRF      bool
	EnableRateLimit bool
	RateLimitRPS    int
	RateLimitBurst  int
	AllowedOrigins  []string
	TrustedProxies  []string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnvString("SERVER_HOST", "localhost"),
			Port:            getEnvInt("SERVER_PORT", 8084),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			CSVFile: getEnvString("CSV_FILE", "data.csv"),
		},
		Logger: LoggerConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
		Security: SecurityConfig{
			EnableCSRF:      getEnvBool("SECURITY_CSRF_ENABLED", true),
			EnableRateLimit: getEnvBool("SECURITY_RATE_LIMIT_ENABLED", true),
			RateLimitRPS:    getEnvInt("SECURITY_RATE_LIMIT_RPS", 100),
			RateLimitBurst:  getEnvInt("SECURITY_RATE_LIMIT_BURST", 10),
			AllowedOrigins:  getEnvStringSlice("SECURITY_ALLOWED_ORIGINS", []string{"http://localhost:8084"}),
			TrustedProxies:  getEnvStringSlice("SECURITY_TRUSTED_PROXIES", []string{"127.0.0.1"}),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}

	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive")
	}

	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive")
	}

	if c.Database.CSVFile == "" {
		return fmt.Errorf("CSV file path cannot be empty")
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.Logger.Level) {
		return fmt.Errorf("invalid log level %q, must be one of: %s", c.Logger.Level, strings.Join(validLogLevels, ", "))
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, c.Logger.Format) {
		return fmt.Errorf("invalid log format %q, must be one of: %s", c.Logger.Format, strings.Join(validLogFormats, ", "))
	}

	if c.Security.RateLimitRPS <= 0 {
		return fmt.Errorf("rate limit RPS must be positive")
	}

	if c.Security.RateLimitBurst <= 0 {
		return fmt.Errorf("rate limit burst must be positive")
	}

	return nil
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
