package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("creates config with defaults", func(t *testing.T) {
		cfg := New()

		if cfg.TransportType != "stdio" {
			t.Errorf("Expected TransportType 'stdio', got %s", cfg.TransportType)
		}

		if cfg.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort 8080, got %d", cfg.HTTPPort)
		}

		if cfg.ServerName != "Coffee Shop Server" {
			t.Errorf("Expected ServerName 'Coffee Shop Server', got %s", cfg.ServerName)
		}

		if cfg.ServerVersion != "1.0.0" {
			t.Errorf("Expected ServerVersion '1.0.0', got %s", cfg.ServerVersion)
		}

		if cfg.RequestTimeout != 30*time.Second {
			t.Errorf("Expected RequestTimeout 30s, got %v", cfg.RequestTimeout)
		}

		if cfg.ShutdownTimeout != 5*time.Second {
			t.Errorf("Expected ShutdownTimeout 5s, got %v", cfg.ShutdownTimeout)
		}

		if cfg.ReadTimeout != 30*time.Second {
			t.Errorf("Expected ReadTimeout 30s, got %v", cfg.ReadTimeout)
		}

		if cfg.WriteTimeout != 30*time.Second {
			t.Errorf("Expected WriteTimeout 30s, got %v", cfg.WriteTimeout)
		}

		if cfg.IdleTimeout != 120*time.Second {
			t.Errorf("Expected IdleTimeout 120s, got %v", cfg.IdleTimeout)
		}
	})
}

func TestParseFlags(t *testing.T) {
	t.Run("parses flags with default values", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{"test"}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TransportType != "stdio" {
			t.Errorf("Expected TransportType 'stdio', got %s", cfg.TransportType)
		}

		if cfg.HTTPPort != 8080 {
			t.Errorf("Expected HTTPPort 8080, got %d", cfg.HTTPPort)
		}

		if cfg.RequestTimeout != 30*time.Second {
			t.Errorf("Expected RequestTimeout 30s, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("parses flags with custom values", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-transport", "http",
			"-port", "9000",
			"-request-timeout", "60s",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TransportType != "http" {
			t.Errorf("Expected TransportType 'http', got %s", cfg.TransportType)
		}

		if cfg.HTTPPort != 9000 {
			t.Errorf("Expected HTTPPort 9000, got %d", cfg.HTTPPort)
		}

		if cfg.RequestTimeout != 60*time.Second {
			t.Errorf("Expected RequestTimeout 60s, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("returns validation error for invalid port", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-port", "99999",
		}
		os.Args = args

		_, err := ParseFlags()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid port: 99999 (must be 1-65535)" {
			t.Errorf("Expected invalid port error, got: %v", err)
		}
	})

	t.Run("returns validation error for invalid timeout", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "0s",
		}
		os.Args = args

		_, err := ParseFlags()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid request timeout: 0s (must be positive)" {
			t.Errorf("Expected invalid timeout error, got: %v", err)
		}
	})

	t.Run("parses negative timeout as zero", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "-5s",
		}
		os.Args = args

		_, err := ParseFlags()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid request timeout: -5s (must be positive)" {
			t.Errorf("Expected invalid timeout error, got: %v", err)
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := New()

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid port - too low", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = 0

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid port: 0 (must be 1-65535)" {
			t.Errorf("Expected invalid port error, got: %v", err)
		}
	})

	t.Run("invalid port - negative", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = -1

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid port: -1 (must be 1-65535)" {
			t.Errorf("Expected invalid port error, got: %v", err)
		}
	})

	t.Run("invalid port - too high", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = 70000

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid port: 70000 (must be 1-65535)" {
			t.Errorf("Expected invalid port error, got: %v", err)
		}
	})

	t.Run("valid port - minimum", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = 1

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid port - maximum", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = 65535

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid request timeout - zero", func(t *testing.T) {
		cfg := New()
		cfg.RequestTimeout = 0

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid request timeout: 0s (must be positive)" {
			t.Errorf("Expected invalid timeout error, got: %v", err)
		}
	})

	t.Run("invalid request timeout - negative", func(t *testing.T) {
		cfg := New()
		cfg.RequestTimeout = -1 * time.Second

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "invalid request timeout: -1s (must be positive)" {
			t.Errorf("Expected invalid timeout error, got: %v", err)
		}
	})

	t.Run("valid request timeout - very small", func(t *testing.T) {
		cfg := New()
		cfg.RequestTimeout = 1 * time.Millisecond

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid request timeout - very large", func(t *testing.T) {
		cfg := New()
		cfg.RequestTimeout = 24 * time.Hour

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("multiple validation errors - returns first", func(t *testing.T) {
		cfg := New()
		cfg.HTTPPort = 0
		cfg.RequestTimeout = -1 * time.Second

		err := cfg.Validate()
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		// Should return the first error encountered (port)
		if err.Error() != "invalid port: 0 (must be 1-65535)" {
			t.Errorf("Expected invalid port error first, got: %v", err)
		}
	})
}

func TestConfig_ParseVariousTimeDurations(t *testing.T) {
	t.Run("parses millisecond duration", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "500ms",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.RequestTimeout != 500*time.Millisecond {
			t.Errorf("Expected RequestTimeout 500ms, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("parses second duration", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "45s",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.RequestTimeout != 45*time.Second {
			t.Errorf("Expected RequestTimeout 45s, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("parses minute duration", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "2m",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.RequestTimeout != 2*time.Minute {
			t.Errorf("Expected RequestTimeout 2m, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("parses hour duration", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-request-timeout", "1h",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.RequestTimeout != 1*time.Hour {
			t.Errorf("Expected RequestTimeout 1h, got %v", cfg.RequestTimeout)
		}
	})
}

func TestConfig_TransportType(t *testing.T) {
	t.Run("accepts stdio transport type", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-transport", "stdio",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TransportType != "stdio" {
			t.Errorf("Expected TransportType 'stdio', got %s", cfg.TransportType)
		}
	})

	t.Run("accepts http transport type", func(t *testing.T) {
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-transport", "http",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TransportType != "http" {
			t.Errorf("Expected TransportType 'http', got %s", cfg.TransportType)
		}
	})

	t.Run("accepts custom transport type", func(t *testing.T) {
		// ParseFlags doesn't validate transport type, just accepts any value
		// Reset flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		args := []string{
			"test",
			"-transport", "custom-transport",
		}
		os.Args = args

		cfg, err := ParseFlags()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TransportType != "custom-transport" {
			t.Errorf("Expected TransportType 'custom-transport', got %s", cfg.TransportType)
		}
	})
}
