package main

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/transport"
)

func TestCreateTransport(t *testing.T) {
	cfg := config.New()

	cfg.TransportType = "stdio"
	tx, err := createTransport(cfg)
	if err != nil {
		t.Fatalf("expected stdio transport, got error: %v", err)
	}
	if _, ok := tx.(*transport.Stdio); !ok {
		t.Fatalf("expected *transport.Stdio, got %T", tx)
	}

	cfg.TransportType = "http"
	tx, err = createTransport(cfg)
	if err != nil {
		t.Fatalf("expected http transport, got error: %v", err)
	}
	if _, ok := tx.(*transport.HTTPTransport); !ok {
		t.Fatalf("expected *transport.HTTPTransport, got %T", tx)
	}

	cfg.TransportType = "invalid"
	_, err = createTransport(cfg)
	if err == nil {
		t.Fatal("expected invalid transport type error")
	}
}

func TestRunWithNilConfig(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("expected run(nil) to fail")
	}
	if !strings.Contains(err.Error(), "failed to create server") {
		t.Fatalf("expected wrapped server creation error, got %v", err)
	}
}

func TestRunWithInvalidTransportConfig(t *testing.T) {
	cfg := config.New()
	cfg.TransportType = "invalid"

	err := run(cfg)
	if err == nil {
		t.Fatal("expected run with invalid transport to fail")
	}
	if !strings.Contains(err.Error(), "failed to create transport") {
		t.Fatalf("expected wrapped transport creation error, got %v", err)
	}
}

func TestRunWithMissingSpecFile(t *testing.T) {
	cfg := config.New()
	cfg.SpecPath = "/path/that/does/not/exist.json"

	err := run(cfg)
	if err == nil {
		t.Fatal("expected run with missing spec file to fail")
	}
	if !strings.Contains(err.Error(), "failed to load spec") {
		t.Fatalf("expected wrapped spec load error, got %v", err)
	}
}

func TestRunWithStdioEOF(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	cfg := config.New()
	cfg.TransportType = "stdio"
	cfg.RequestTimeout = 50 * time.Millisecond
	specPath := createTestSpecFile(t)
	cfg.SpecPath = specPath

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdin writer: %v", err)
	}

	err = run(cfg)
	if err != nil {
		t.Fatalf("expected run to exit cleanly on EOF stdin, got %v", err)
	}
}

func createTestSpecFile(t *testing.T) string {
	t.Helper()

	path := t.TempDir() + "/mcp-spec.json"
	content := `{
  "schemaVersion": "v1",
  "server": {"name": "Template MCP", "version": "1.0.0"},
  "runtime": {"transportType": "stdio"},
  "items": [
    {"name": "Item A", "price": 5, "category": "starter", "description": "First item"}
  ],
  "tools": [
    {"mode": "list_items", "name": "listItems", "description": "List items", "inputSchema": {"type": "object", "properties": {}, "required": []}},
    {"mode": "get_item_details", "name": "getItemDetails", "description": "Get item details", "inputSchema": {"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}}
  ],
  "resources": [
    {"mode": "catalog_items", "uri": "catalog://items", "name": "catalog"}
  ],
  "prompts": [
    {"mode": "plan_recommendation", "name": "planRecommendation", "description": "Plan prompt", "arguments": [{"name": "budget", "description": "Budget", "required": false}], "template": "Plan for a team%s%s"},
    {"mode": "item_brief", "name": "itemBrief", "description": "Brief prompt", "arguments": [{"name": "item_name", "description": "Item", "required": true}], "template": "Brief for %s"}
  ]
}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test spec file: %v", err)
	}

	return path
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		parseErr error
		runErr   error
		wantCode int
		wantLog  string
	}{
		{name: "parse error", parseErr: errors.New("bad flags"), wantCode: 1, wantLog: "configuration error"},
		{name: "run error", runErr: errors.New("boom"), wantCode: 1, wantLog: "server error"},
		{name: "success", wantCode: 0, wantLog: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf strings.Builder
			parse := func() (*config.Config, error) {
				if tt.parseErr != nil {
					return nil, tt.parseErr
				}
				return config.New(), nil
			}
			runServer := func(cfg *config.Config) error {
				return tt.runErr
			}

			code := execute(parse, runServer, &logBuf)
			if code != tt.wantCode {
				t.Fatalf("expected code %d, got %d", tt.wantCode, code)
			}

			logs := logBuf.String()
			if tt.wantLog != "" && !strings.Contains(logs, tt.wantLog) {
				t.Fatalf("expected logs to contain %q, got %q", tt.wantLog, logs)
			}
			if tt.wantLog == "" && logs != "" {
				t.Fatalf("expected no logs for success, got %q", logs)
			}
		})
	}
}
