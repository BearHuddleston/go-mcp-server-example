package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cfg := New()

	if cfg == nil {
		t.Fatal("New() returned nil config")
	}

	t.Run("TransportType", func(t *testing.T) {
		if cfg.TransportType != "stdio" {
			t.Errorf("Expected 'stdio', got %s", cfg.TransportType)
		}
	})

	t.Run("HTTPPort", func(t *testing.T) {
		if cfg.HTTPPort != 8080 {
			t.Errorf("Expected 8080, got %d", cfg.HTTPPort)
		}
	})

	t.Run("ServerName", func(t *testing.T) {
		if cfg.ServerName != "Coffee Shop Server" {
			t.Errorf("Expected 'Coffee Shop Server', got %s", cfg.ServerName)
		}
	})

	t.Run("ServerVersion", func(t *testing.T) {
		if cfg.ServerVersion != "1.0.0" {
			t.Errorf("Expected '1.0.0', got %s", cfg.ServerVersion)
		}
	})

	t.Run("RequestTimeout", func(t *testing.T) {
		if cfg.RequestTimeout != 30*time.Second {
			t.Errorf("Expected 30s, got %v", cfg.RequestTimeout)
		}
	})

	t.Run("ShutdownTimeout", func(t *testing.T) {
		if cfg.ShutdownTimeout != 5*time.Second {
			t.Errorf("Expected 5s, got %v", cfg.ShutdownTimeout)
		}
	})

	t.Run("ReadTimeout", func(t *testing.T) {
		if cfg.ReadTimeout != 30*time.Second {
			t.Errorf("Expected 30s, got %v", cfg.ReadTimeout)
		}
	})

	t.Run("WriteTimeout", func(t *testing.T) {
		if cfg.WriteTimeout != 30*time.Second {
			t.Errorf("Expected 30s, got %v", cfg.WriteTimeout)
		}
	})

	t.Run("IdleTimeout", func(t *testing.T) {
		if cfg.IdleTimeout != 120*time.Second {
			t.Errorf("Expected 120s, got %v", cfg.IdleTimeout)
		}
	})

	t.Run("AllowedOrigins", func(t *testing.T) {
		if len(cfg.AllowedOrigins) != 2 {
			t.Errorf("Expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
		}
		if cfg.AllowedOrigins[0] != "http://localhost:*" {
			t.Errorf("Expected allowed origin 'http://localhost:*', got %s", cfg.AllowedOrigins[0])
		}
	})
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name:    "default values",
			args:    []string{},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.TransportType != "stdio" {
					t.Errorf("Expected TransportType 'stdio', got %s", cfg.TransportType)
				}
				if cfg.HTTPPort != 8080 {
					t.Errorf("Expected HTTPPort 8080, got %d", cfg.HTTPPort)
				}
				if cfg.RequestTimeout != 30*time.Second {
					t.Errorf("Expected RequestTimeout 30s, got %v", cfg.RequestTimeout)
				}
			},
		},
		{
			name:    "custom values",
			args:    []string{"-transport", "http", "-port", "9000", "-request-timeout", "60s"},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.TransportType != "http" {
					t.Errorf("Expected TransportType 'http', got %s", cfg.TransportType)
				}
				if cfg.HTTPPort != 9000 {
					t.Errorf("Expected HTTPPort 9000, got %d", cfg.HTTPPort)
				}
				if cfg.RequestTimeout != 60*time.Second {
					t.Errorf("Expected RequestTimeout 60s, got %v", cfg.RequestTimeout)
				}
			},
		},
		{
			name:    "allowed origins",
			args:    []string{"-allowed-origins", "https://example.com,https://api.example.com"},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.AllowedOrigins) != 2 {
					t.Errorf("Expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
				}
				if cfg.AllowedOrigins[0] != "https://example.com" {
					t.Errorf("Expected allowed origin 'https://example.com', got %s", cfg.AllowedOrigins[0])
				}
			},
		},
		{
			name:    "invalid timeout",
			args:    []string{"-request-timeout", "0s"},
			wantErr: true,
			validate: func(t *testing.T, cfg *Config) {
			},
		},
		{
			name:    "invalid port",
			args:    []string{"-port", "0"},
			wantErr: true,
			validate: func(t *testing.T, cfg *Config) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = append([]string{"test"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			cfg, err := ParseFlags()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
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
				HTTPPort:       8080,
				RequestTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			cfg: &Config{
				HTTPPort:       0,
				RequestTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &Config{
				HTTPPort:       70000,
				RequestTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid request timeout",
			cfg: &Config{
				HTTPPort:       8080,
				RequestTimeout: 0,
			},
			wantErr: true,
		},
		{
			name: "negative request timeout",
			cfg: &Config{
				HTTPPort:       8080,
				RequestTimeout: -30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfig_ParseVariousTimeDurations(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		timeout time.Duration
	}{
		{
			name:    "millisecond duration",
			args:    []string{"-request-timeout", "500ms"},
			timeout: 500 * time.Millisecond,
		},
		{
			name:    "second duration",
			args:    []string{"-request-timeout", "45s"},
			timeout: 45 * time.Second,
		},
		{
			name:    "minute duration",
			args:    []string{"-request-timeout", "2m"},
			timeout: 2 * time.Minute,
		},
		{
			name:    "hour duration",
			args:    []string{"-request-timeout", "1h"},
			timeout: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = append([]string{"test"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			cfg, err := ParseFlags()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg.RequestTimeout != tt.timeout {
				t.Errorf("Expected RequestTimeout %v, got %v", tt.timeout, cfg.RequestTimeout)
			}
		})
	}
}

func TestConfig_TransportType(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		transport string
	}{
		{
			name:      "stdio transport",
			args:      []string{"-transport", "stdio"},
			transport: "stdio",
		},
		{
			name:      "http transport",
			args:      []string{"-transport", "http"},
			transport: "http",
		},
		{
			name:      "custom transport",
			args:      []string{"-transport", "custom-transport"},
			transport: "custom-transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = append([]string{"test"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			cfg, err := ParseFlags()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg.TransportType != tt.transport {
				t.Errorf("Expected TransportType '%s', got %s", tt.transport, cfg.TransportType)
			}
		})
	}
}
