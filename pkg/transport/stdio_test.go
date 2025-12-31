package transport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

type mockServer struct{}

func (m *mockServer) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return &mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    map[string]any{},
		ServerInfo: mcp.ServerInfo{
			Name:    "Test Server",
			Version: "1.0.0",
		},
	}, nil
}

func (m *mockServer) HandleRequest(ctx context.Context, req mcp.Request) error {
	sender := ctx.Value(mcp.ResponseSenderKey).(mcp.ResponseSender)

	response := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      req.ID,
		Result: map[string]any{
			"test": "response",
		},
	}

	return sender.SendResponse(response)
}

func TestNewStdio(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *config.Config
		check func(*testing.T, *Stdio)
	}{
		{
			name: "with config",
			cfg: &config.Config{
				RequestTimeout: 45 * time.Second,
			},
			check: func(t *testing.T, s *Stdio) {
				if s.config == nil {
					t.Error("Expected config to be set")
				}
				if s.config.RequestTimeout != 45*time.Second {
					t.Errorf("Expected RequestTimeout 45s, got %v", s.config.RequestTimeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdio := NewStdio(tt.cfg)
			tt.check(t, stdio)
		})
	}
}

func TestStdio_UsesConfigTimeout(t *testing.T) {
	tests := []struct {
		name           string
		requestTimeout time.Duration
		expectTimeout  time.Duration
	}{
		{
			name:           "45 second timeout",
			requestTimeout: 45 * time.Second,
			expectTimeout:  45 * time.Second,
		},
		{
			name:           "60 second timeout",
			requestTimeout: 60 * time.Second,
			expectTimeout:  60 * time.Second,
		},
		{
			name:           "10 second timeout",
			requestTimeout: 10 * time.Second,
			expectTimeout:  10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RequestTimeout: tt.requestTimeout,
			}

			stdio := NewStdio(cfg)

			if stdio.config.RequestTimeout != tt.expectTimeout {
				t.Errorf("Expected RequestTimeout %v, got %v", tt.expectTimeout, stdio.config.RequestTimeout)
			}
		})
	}
}

func TestStdio_handleMessage(t *testing.T) {
	tests := []struct {
		name      string
		request   string
		checkFunc func(*testing.T, *Stdio, string, error)
	}{
		{
			name: "valid request",
			request: `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "test"
			}`,
			checkFunc: func(t *testing.T, s *Stdio, input string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			},
		},
		{
			name: "notification - no ID",
			request: `{
				"jsonrpc": "2.0",
				"method": "test"
			}`,
			checkFunc: func(t *testing.T, s *Stdio, input string, err error) {
				if err != nil {
					t.Errorf("Expected no error for notification, got %v", err)
				}
			},
		},
		{
			name:    "invalid JSON",
			request: `{invalid json}`,
			checkFunc: func(t *testing.T, s *Stdio, input string, err error) {
				if err != nil {
					t.Errorf("Expected no error from parsing (error is logged), got %v", err)
				}
			},
		},
		{
			name: "wrong JSON-RPC version",
			request: `{
				"jsonrpc": "1.0",
				"id": 1,
				"method": "test"
			}`,
			checkFunc: func(t *testing.T, s *Stdio, input string, err error) {
				if err != nil {
					t.Errorf("Expected no error (logged), got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RequestTimeout: 30 * time.Second,
			}
			stdio := NewStdio(cfg)
			ctx := context.Background()
			mockSrv := &mockServer{}

			err := stdio.handleMessage(ctx, mockSrv, tt.request)
			tt.checkFunc(t, stdio, tt.request, err)
		})
	}
}

func TestStdio_Stop(t *testing.T) {
	cfg := &config.Config{
		RequestTimeout: 30 * time.Second,
	}
	stdio := NewStdio(cfg)

	err := stdio.Stop()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStdio_sendParseError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		parseErr error
		checkErr func(*testing.T, error)
	}{
		{
			name:     "valid JSON with ID and real parse error",
			input:    `{"jsonrpc": "2.0", "id": 123, "method": "test"}`,
			parseErr: json.Unmarshal([]byte(`invalid`), &struct{}{}),
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error from sendParseError, got %v", err)
				}
			},
		},
		{
			name:     "completely invalid JSON",
			input:    `not json at all`,
			parseErr: json.Unmarshal([]byte(`not json`), &struct{}{}),
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error from sendParseError, got %v", err)
				}
			},
		},
		{
			name:     "empty input",
			input:    "",
			parseErr: json.Unmarshal([]byte(``), &struct{}{}),
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error from sendParseError, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RequestTimeout: 30 * time.Second,
			}
			stdio := NewStdio(cfg)

			err := stdio.sendParseError(tt.input, tt.parseErr)

			tt.checkErr(t, err)
		})
	}
}
