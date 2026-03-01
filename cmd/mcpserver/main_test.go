package main

import (
	"strings"
	"testing"

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
