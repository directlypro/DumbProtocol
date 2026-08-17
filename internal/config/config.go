package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env            string
	Host           string
	Port           int
	DatabaseDriver string
	DatabaseURL    string
	TOTPIssuer     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// Load reads settings from environment variables and .env file if present, with defaults.
func Load() (*Config, error) {
	// Attempt to load .env, ignore error if missing
	_ = godotenv.Load()

	cfg := &Config{
		Env:            getEnv("APP_ENV", "development"),
		Host:           getEnv("HOST", "127.0.0.1"),
		Port:           getEnvAsInt("PORT", 8080),
		DatabaseDriver: getEnv("DATABASE_DRIVER", "postgres"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/dumbprotocol?sslmode=disable"),
		TOTPIssuer:     getEnv("TOTP_ISSUER", "DumbProtocol"),
		ReadTimeout:    getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:   getEnvAsDuration("WRITE_TIMEOUT", 15*time.Second),
	}

	// Legacy fallback: if URL or DATABASE env vars are set
	if dbURL := os.Getenv("DATABASE"); dbURL != "" && os.Getenv("DATABASE_URL") == "" {
		cfg.DatabaseURL = dbURL
	}
	if legacyURL := os.Getenv("URL"); legacyURL != "" && os.Getenv("PORT") == "" {
		// format 127.0.0.1:3306 or similar
		cfg.Host, cfg.Port = parseHostPort(legacyURL, cfg.Host, cfg.Port)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.DatabaseDriver != "sqlite" && c.DatabaseDriver != "postgres" && c.DatabaseDriver != "postgresql" {
		return fmt.Errorf("unsupported database driver: %s", c.DatabaseDriver)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL cannot be empty")
	}
	if c.TOTPIssuer == "" {
		c.TOTPIssuer = "DumbProtocol"
	}
	return nil
}

func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if d, err := time.ParseDuration(valStr); err == nil {
		return d
	}
	return defaultVal
}

func parseHostPort(raw string, defaultHost string, defaultPort int) (string, int) {
	// simple helper for legacy "127.0.0.1:8080" format
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == ':' {
			host := raw[:i]
			if host == "" {
				host = defaultHost
			}
			if port, err := strconv.Atoi(raw[i+1:]); err == nil {
				return host, port
			}
			break
		}
	}
	return defaultHost, defaultPort
}
