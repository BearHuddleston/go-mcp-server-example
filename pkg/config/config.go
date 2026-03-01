// Package config provides configuration management for the MCP server.
package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds all configuration for the MCP server
type Config struct {
	// Transport settings
	TransportType string
	HTTPPort      int

	// Server settings
	ServerName    string
	ServerVersion string

	// Timeouts
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration

	// HTTP settings
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	AllowedOrigins []string
}

// New creates a new configuration with defaults
func New() *Config {
	return &Config{
		TransportType:   "stdio",
		HTTPPort:        8080,
		ServerName:      "Coffee Shop Server",
		ServerVersion:   "1.0.0",
		RequestTimeout:  30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		AllowedOrigins:  []string{"http://localhost:*", "http://127.0.0.1:*"},
	}
}

// ParseFlags parses command line flags and returns a config
func ParseFlags() (*Config, error) {
	cfg := New()

	transportType := flag.String("transport", cfg.TransportType, "Transport type: stdio or http")
	port := flag.Int("port", cfg.HTTPPort, "Port for HTTP transport (ignored for stdio)")
	requestTimeout := flag.Duration("request-timeout", cfg.RequestTimeout, "Request timeout duration")
	allowedOrigins := flag.String("allowed-origins", "", "Comma-separated list of allowed CORS origins (e.g., https://example.com,https://api.example.com)")

	flag.Parse()

	cfg.TransportType = *transportType
	cfg.HTTPPort = *port
	cfg.RequestTimeout = *requestTimeout
	if *allowedOrigins != "" {
		cfg.AllowedOrigins = parseAllowedOrigins(*allowedOrigins)
	}

	return cfg, cfg.Validate()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.HTTPPort)
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("invalid request timeout: %v (must be positive)", c.RequestTimeout)
	}

	return nil
}

func parseAllowedOrigins(value string) []string {
	origins := strings.Split(value, ",")
	normalized := make([]string, 0, len(origins))

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
