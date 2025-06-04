// Package transport provides MCP transport implementations.
package transport

import (
	"context"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// Transport defines the interface for MCP transport mechanisms.
type Transport interface {
	// Start begins listening for requests on this transport.
	Start(ctx context.Context, server mcp.Server) error
	// Stop gracefully shuts down the transport.
	Stop() error
}
