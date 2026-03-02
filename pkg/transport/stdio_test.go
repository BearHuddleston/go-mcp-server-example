package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/BearHuddleston/mcp-server-template/pkg/config"
	"github.com/BearHuddleston/mcp-server-template/pkg/mcp"
)

type mockServer struct{}

type countingServer struct {
	calls int
}

type failingReader struct {
	err error
}

type failingWriter struct{}

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

func (m *countingServer) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return (&mockServer{}).Initialize(ctx)
}

func (m *countingServer) HandleRequest(ctx context.Context, req mcp.Request) error {
	m.calls++
	return (&mockServer{}).HandleRequest(ctx, req)
}

func (r *failingReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func (f *failingWriter) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
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
		{
			name: "with nil config",
			cfg:  nil,
			check: func(t *testing.T, s *Stdio) {
				if s.config == nil {
					t.Error("Expected config to be defaulted, got nil")
				}
				if s.config.RequestTimeout != 30*time.Second {
					t.Errorf("Expected default RequestTimeout 30s, got %v", s.config.RequestTimeout)
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

type slowMockServer struct {
	delay time.Duration
}

func (m *slowMockServer) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return &mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    map[string]any{},
		ServerInfo: mcp.ServerInfo{
			Name:    "Test Server",
			Version: "1.0.0",
		},
	}, nil
}

func (m *slowMockServer) HandleRequest(ctx context.Context, req mcp.Request) error {
	select {
	case <-time.After(m.delay):
		sender := ctx.Value(mcp.ResponseSenderKey).(mcp.ResponseSender)
		response := mcp.Response{
			JSONRPC: mcp.JSONRPCVersion,
			ID:      req.ID,
			Result:  map[string]any{"test": "response"},
		}
		return sender.SendResponse(response)
	case <-ctx.Done():
		return nil
	}
}

func TestStdio_TimeoutApplied(t *testing.T) {
	tests := []struct {
		name          string
		configTimeout time.Duration
		serverDelay   time.Duration
		expectTimeout bool
	}{
		{
			name:          "sufficient timeout - no cancellation",
			configTimeout: 100 * time.Millisecond,
			serverDelay:   50 * time.Millisecond,
			expectTimeout: false,
		},
		{
			name:          "insufficient timeout - context cancelled",
			configTimeout: 50 * time.Millisecond,
			serverDelay:   200 * time.Millisecond,
			expectTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RequestTimeout: tt.configTimeout,
			}

			stdio := NewStdio(cfg)
			slowSrv := &slowMockServer{delay: tt.serverDelay}

			ctx := context.Background()
			start := time.Now()

			req := `{"jsonrpc":"2.0","id":1,"method":"test"}`
			err := stdio.handleMessage(ctx, slowSrv, req)

			elapsed := time.Since(start)

			if tt.expectTimeout {
				if elapsed > tt.configTimeout+10*time.Millisecond {
					t.Errorf("Expected request to timeout after ~%v, but took %v", tt.configTimeout, elapsed)
				}
				if err != nil {
					t.Errorf("Expected no error (context handles cancellation), got %v", err)
				}
			} else {
				if elapsed < tt.serverDelay {
					t.Errorf("Expected request to complete after server delay %v, but completed in %v", tt.serverDelay, elapsed)
				}
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
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

func TestStdio_Start(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		cfg := &config.Config{RequestTimeout: 30 * time.Second}
		stdio := NewStdio(cfg)
		stdio.input = strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"test\"}\n")
		stdio.output = &bytes.Buffer{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := stdio.Start(ctx, &mockServer{})
		if err != nil {
			t.Fatalf("expected nil on canceled context, got %v", err)
		}
	})

	t.Run("eof exits cleanly", func(t *testing.T) {
		cfg := &config.Config{RequestTimeout: 30 * time.Second}
		stdio := NewStdio(cfg)
		stdio.input = strings.NewReader("")
		stdio.output = &bytes.Buffer{}

		err := stdio.Start(context.Background(), &mockServer{})
		if err != nil {
			t.Fatalf("expected nil on EOF, got %v", err)
		}
	})

	t.Run("blank line skipped and valid message handled", func(t *testing.T) {
		cfg := &config.Config{RequestTimeout: 30 * time.Second}
		stdio := NewStdio(cfg)
		stdio.input = strings.NewReader("\n{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"test\"}\n")
		var out bytes.Buffer
		stdio.output = &out

		srv := &countingServer{}
		err := stdio.Start(context.Background(), srv)
		if err != nil {
			t.Fatalf("expected nil for normal scan completion, got %v", err)
		}
		if srv.calls != 1 {
			t.Fatalf("expected one handled request, got %d", srv.calls)
		}
		if !strings.Contains(out.String(), `"test":"response"`) {
			t.Fatalf("expected response output, got %q", out.String())
		}
	})

	t.Run("scanner read error is returned", func(t *testing.T) {
		cfg := &config.Config{RequestTimeout: 30 * time.Second}
		stdio := NewStdio(cfg)
		stdio.input = &failingReader{err: errors.New("read failure")}
		stdio.output = &bytes.Buffer{}

		err := stdio.Start(context.Background(), &mockServer{})
		if err == nil {
			t.Fatal("expected scanner read error")
		}
		if !strings.Contains(err.Error(), "read failure") {
			t.Fatalf("expected read failure error, got %v", err)
		}
	})
}

func TestStdoutSender(t *testing.T) {
	t.Run("SendError writes JSON error response", func(t *testing.T) {
		var out bytes.Buffer
		sender := &StdoutSender{writer: &out}

		err := sender.SendError("id-1", mcp.ErrorCodeInternalError, "boom", nil)
		if err != nil {
			t.Fatalf("SendError failed: %v", err)
		}

		line := strings.TrimSpace(out.String())
		if !strings.Contains(line, "\"error\"") || !strings.Contains(line, "boom") {
			t.Fatalf("expected serialized error JSON, got %q", line)
		}
	})

	t.Run("SendResponse returns write errors", func(t *testing.T) {
		sender := &StdoutSender{writer: &failingWriter{}}
		err := sender.SendResponse(mcp.Response{JSONRPC: mcp.JSONRPCVersion, ID: 1, Result: map[string]any{"ok": true}})
		if err == nil {
			t.Fatal("expected write failure error")
		}
	})
}

func TestStdio_sendParseError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkErr func(*testing.T, error)
	}{
		{
			name:  "completely invalid JSON",
			input: `not json at all`,
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error from sendParseError, got %v", err)
				}
			},
		},
		{
			name:  "malformed JSON with syntax error",
			input: `{"jsonrpc": "2.0", "id": 123, "method": "test"`,
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error from sendParseError, got %v", err)
				}
			},
		},
		{
			name:  "empty input",
			input: "",
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
			stdio.output = &bytes.Buffer{}

			parseErr := json.Unmarshal([]byte(tt.input), &struct{}{})
			err := stdio.sendParseError(tt.input, parseErr)

			tt.checkErr(t, err)
		})
	}
}
