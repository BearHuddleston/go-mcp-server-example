package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/BearHuddleston/mcp-server-template/internal/server"
	"github.com/BearHuddleston/mcp-server-template/pkg/config"
	"github.com/BearHuddleston/mcp-server-template/pkg/handlers"
	"github.com/BearHuddleston/mcp-server-template/pkg/spec"
	"github.com/BearHuddleston/mcp-server-template/pkg/transport"
)

var (
	parseFlagsFunc = config.ParseFlags
	runFunc        = run
	exitFunc       = os.Exit
)

func main() {
	if code := execute(parseFlagsFunc, runFunc, os.Stderr); code != 0 {
		exitFunc(code)
	}
}

func execute(parseFlags func() (*config.Config, error), runServer func(*config.Config) error, stderr io.Writer) int {
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := parseFlags()
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}

	if err := runServer(cfg); err != nil {
		slog.Error("server error", "error", err)
		return 1
	}

	return 0
}

// run starts and runs the MCP server with the given configuration
func run(cfg *config.Config) error {
	catalogHandler := handlers.NewCatalog()
	if cfg != nil && cfg.SpecPath != "" {
		sp, err := spec.LoadFile(cfg.SpecPath)
		if err != nil {
			return fmt.Errorf("failed to load spec: %w", err)
		}

		catalogHandler, err = handlers.NewCatalogFromSpec(sp)
		if err != nil {
			return fmt.Errorf("failed to create catalog from spec: %w", err)
		}
	}

	mcpServer, err := server.New(cfg, catalogHandler, catalogHandler, catalogHandler)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	transport, err := createTransport(cfg)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("received shutdown signal; stopping server")
		cancel()
	}()

	if err := transport.Start(ctx, mcpServer); err != nil {
		return fmt.Errorf("transport start failed: %w", err)
	}

	return nil
}

// createTransport creates the appropriate transport based on configuration
func createTransport(cfg *config.Config) (transport.Transport, error) {
	switch strings.ToLower(cfg.TransportType) {
	case "stdio":
		return transport.NewStdio(cfg), nil
	case "http":
		return transport.NewHTTP(cfg), nil
	default:
		return nil, fmt.Errorf("invalid transport type: %s (must be 'stdio' or 'http')", cfg.TransportType)
	}
}
