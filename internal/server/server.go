// Package server provides the internal MCP server implementation.
package server

import (
	"context"
	"fmt"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// Server implements the core MCP server logic
type Server struct {
	toolHandler     mcp.ToolHandler
	resourceHandler mcp.ResourceHandler
	promptHandler   mcp.PromptHandler
	serverInfo      mcp.ServerInfo
}

// New creates a new MCP server with the given handlers
func New(cfg *config.Config, toolHandler mcp.ToolHandler, resourceHandler mcp.ResourceHandler, promptHandler mcp.PromptHandler) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if toolHandler == nil {
		return nil, fmt.Errorf("toolHandler cannot be nil")
	}
	if resourceHandler == nil {
		return nil, fmt.Errorf("resourceHandler cannot be nil")
	}
	if promptHandler == nil {
		return nil, fmt.Errorf("promptHandler cannot be nil")
	}

	return &Server{
		toolHandler:     toolHandler,
		resourceHandler: resourceHandler,
		promptHandler:   promptHandler,
		serverInfo: mcp.ServerInfo{
			Name:    cfg.ServerName,
			Version: cfg.ServerVersion,
		},
	}, nil
}

// Initialize handles the MCP initialization handshake
func (s *Server) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return &mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities: map[string]any{
			"tools":     map[string]bool{"listChanged": true},
			"resources": map[string]bool{"listChanged": true},
			"prompts":   map[string]bool{"listChanged": true},
		},
		ServerInfo: s.serverInfo,
	}, nil
}

// HandleRequest processes a JSON-RPC request
func (s *Server) HandleRequest(ctx context.Context, req mcp.Request) error {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req.ID)
	case "tools/list":
		return s.handleToolsList(ctx, req.ID)
	case "tools/call":
		return s.handleToolsCall(ctx, req.ID, req)
	case "resources/list":
		return s.handleResourcesList(ctx, req.ID)
	case "resources/read":
		return s.handleResourcesRead(ctx, req.ID, req)
	case "prompts/list":
		return s.handlePromptsList(ctx, req.ID)
	case "prompts/get":
		return s.handlePromptsGet(ctx, req.ID, req)
	case "ping":
		return s.handlePing(ctx, req.ID)
	default:
		return s.sendError(ctx, req.ID, mcp.ErrorCodeMethodNotFound, fmt.Sprintf("Method %s not found", req.Method), nil)
	}
}

// Helper methods for sending responses
func (s *Server) sendResponse(ctx context.Context, id any, result any) error {
	response := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
	return s.sendResponseDirect(ctx, response)
}

func (s *Server) sendError(ctx context.Context, id any, code int, message string, data any) error {
	if sender := ctx.Value(mcp.ResponseSenderKey); sender != nil {
		if rs, ok := sender.(mcp.ResponseSender); ok {
			return rs.SendError(id, code, message, data)
		}
	}
	// This shouldn't happen in normal operation
	return fmt.Errorf("no response sender in context")
}

func (s *Server) sendResponseDirect(ctx context.Context, response mcp.Response) error {
	if sender := ctx.Value(mcp.ResponseSenderKey); sender != nil {
		if rs, ok := sender.(mcp.ResponseSender); ok {
			return rs.SendResponse(response)
		}
	}
	// This shouldn't happen in normal operation
	return fmt.Errorf("no response sender in context")
}

// Request handlers
func (s *Server) handleInitialize(ctx context.Context, id any) error {
	result, err := s.Initialize(ctx)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInternalError, "Failed to initialize", err.Error())
	}
	return s.sendResponse(ctx, id, result)
}

func (s *Server) handleToolsList(ctx context.Context, id any) error {
	tools, err := s.toolHandler.ListTools(ctx)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInternalError, "Failed to list tools", err.Error())
	}
	return s.sendResponse(ctx, id, map[string][]mcp.Tool{"tools": tools})
}

func (s *Server) handleToolsCall(ctx context.Context, id any, req mcp.Request) error {
	params, err := s.parseToolCallParams(req.Params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, "Invalid tool call parameters", err.Error())
	}

	response, err := s.toolHandler.CallTool(ctx, params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, fmt.Sprintf("Tool call failed: %s", err.Error()), nil)
	}
	return s.sendResponse(ctx, id, response)
}

func (s *Server) handleResourcesList(ctx context.Context, id any) error {
	resources, err := s.resourceHandler.ListResources(ctx)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInternalError, "Failed to list resources", err.Error())
	}
	return s.sendResponse(ctx, id, map[string][]mcp.Resource{"resources": resources})
}

func (s *Server) handleResourcesRead(ctx context.Context, id any, req mcp.Request) error {
	params, err := s.parseResourceParams(req.Params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, "Invalid resource read parameters", err.Error())
	}

	response, err := s.resourceHandler.ReadResource(ctx, params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, fmt.Sprintf("Resource read failed: %s", err.Error()), nil)
	}
	return s.sendResponse(ctx, id, response)
}

func (s *Server) handlePromptsList(ctx context.Context, id any) error {
	prompts, err := s.promptHandler.ListPrompts(ctx)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInternalError, "Failed to list prompts", err.Error())
	}
	return s.sendResponse(ctx, id, map[string][]mcp.Prompt{"prompts": prompts})
}

func (s *Server) handlePromptsGet(ctx context.Context, id any, req mcp.Request) error {
	params, err := s.parsePromptParams(req.Params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, "Invalid prompt parameters", err.Error())
	}

	response, err := s.promptHandler.GetPrompt(ctx, params)
	if err != nil {
		return s.sendError(ctx, id, mcp.ErrorCodeInvalidParams, fmt.Sprintf("Prompt call failed: %s", err.Error()), nil)
	}
	return s.sendResponse(ctx, id, response)
}

func (s *Server) handlePing(ctx context.Context, id any) error {
	return s.sendResponse(ctx, id, map[string]any{})
}

// Parameter parsing helpers
func (s *Server) parseToolCallParams(params any) (mcp.ToolCallParams, error) {
	if params == nil {
		return mcp.ToolCallParams{}, fmt.Errorf("params cannot be nil")
	}
	
	// Convert params to map
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return mcp.ToolCallParams{}, fmt.Errorf("params must be an object")
	}
	
	// Extract name
	name, ok := paramsMap["name"].(string)
	if !ok {
		return mcp.ToolCallParams{}, fmt.Errorf("name parameter is required and must be a string")
	}
	
	// Extract arguments
	args := make(map[string]any)
	if arguments, exists := paramsMap["arguments"]; exists {
		if argsMap, ok := arguments.(map[string]any); ok {
			args = argsMap
		}
	}
	
	return mcp.ToolCallParams{
		Name:      name,
		Arguments: args,
	}, nil
}

func (s *Server) parseResourceParams(params any) (mcp.ResourceParams, error) {
	if params == nil {
		return mcp.ResourceParams{}, fmt.Errorf("params cannot be nil")
	}
	
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return mcp.ResourceParams{}, fmt.Errorf("params must be an object")
	}
	
	uri, ok := paramsMap["uri"].(string)
	if !ok {
		return mcp.ResourceParams{}, fmt.Errorf("uri parameter is required and must be a string")
	}
	
	return mcp.ResourceParams{URI: uri}, nil
}

func (s *Server) parsePromptParams(params any) (mcp.PromptParams, error) {
	if params == nil {
		return mcp.PromptParams{}, fmt.Errorf("params cannot be nil")
	}
	
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return mcp.PromptParams{}, fmt.Errorf("params must be an object")
	}
	
	name, ok := paramsMap["name"].(string)
	if !ok {
		return mcp.PromptParams{}, fmt.Errorf("name parameter is required and must be a string")
	}
	
	args := make(map[string]any)
	if arguments, exists := paramsMap["arguments"]; exists {
		if argsMap, ok := arguments.(map[string]any); ok {
			args = argsMap
		}
	}
	
	return mcp.PromptParams{
		Name:      name,
		Arguments: args,
	}, nil
}
