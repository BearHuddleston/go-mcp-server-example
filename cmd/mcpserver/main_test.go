package main

import (
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

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdin writer: %v", err)
	}

	err = run(cfg)
	if err != nil {
		t.Fatalf("expected run to exit cleanly on EOF stdin, got %v", err)
	}
}
