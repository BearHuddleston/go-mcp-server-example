// Package mcp provides core Model Context Protocol types and interfaces.
package mcp

import (
	"context"
)

// Constants for MCP protocol
const (
	ProtocolVersion = "2025-03-26"
	JSONRPCVersion  = "2.0"
)

// JSON-RPC 2.0 error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// Core MCP types
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResponse struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
}

// JSON-RPC 2.0 message types
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      any    `json:"id"` // string or number, MUST NOT be null per MCP spec
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"` // Must match request ID, string or number
	Result  any            `json:"result,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Core interfaces

// Server defines the core MCP server interface.
type Server interface {
	// Initialize handles the MCP initialization handshake.
	Initialize(ctx context.Context) (*InitializeResponse, error)
	// HandleRequest processes a JSON-RPC request.
	HandleRequest(ctx context.Context, req Request) error
}

// ToolHandler defines the interface for handling MCP tool operations.
type ToolHandler interface {
	// ListTools returns all available tools.
	ListTools(ctx context.Context) ([]Tool, error)
	// CallTool executes a tool with the given parameters.
	CallTool(ctx context.Context, params ToolCallParams) (ToolResponse, error)
}

// ResourceHandler defines the interface for handling MCP resource operations.
type ResourceHandler interface {
	// ListResources returns all available resources.
	ListResources(ctx context.Context) ([]Resource, error)
	// ReadResource reads the content of a specific resource.
	ReadResource(ctx context.Context, params ResourceParams) (ResourceResponse, error)
}

// PromptHandler defines the interface for handling MCP prompt operations.
type PromptHandler interface {
	// ListPrompts returns all available prompts.
	ListPrompts(ctx context.Context) ([]Prompt, error)
	// GetPrompt generates a prompt with the given parameters.
	GetPrompt(ctx context.Context, params PromptParams) (PromptResponse, error)
}

// ResponseSender defines the interface for sending responses back to clients.
type ResponseSender interface {
	// SendResponse sends a successful response.
	SendResponse(response Response) error
	// SendError sends an error response.
	SendError(id any, code int, message string, data any) error
}

// Context keys for dependency injection
type contextKey string

const ResponseSenderKey contextKey = "responseSender"
const SessionIDKey contextKey = "sessionID"