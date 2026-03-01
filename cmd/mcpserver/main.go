// MCP Coffee Server - A Model Context Protocol server for coffee shop operations
//
// This application provides tools, resources, and prompts for a coffee shop
// through the Model Context Protocol (MCP). It supports both stdio and HTTP
// transports for integration with various MCP clients.
//
// Usage:
//
//	mcpserver [flags]
//
// Flags:
//
//	-transport string: Transport type (stdio|http) (default "stdio")
//	-port int: HTTP port (default 8080)
//	-request-timeout duration: Request timeout (default 30s)
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

	"github.com/BearHuddleston/mcp-server-example/internal/server"
	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/handlers"
	"github.com/BearHuddleston/mcp-server-example/pkg/transport"
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
	coffeeHandler := handlers.NewCoffee()

	mcpServer, err := server.New(cfg, coffeeHandler, coffeeHandler, coffeeHandler)
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
