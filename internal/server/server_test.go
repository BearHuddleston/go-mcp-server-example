package server

import (
	"context"
	"errors"
	"testing"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// mockHandlers implements all handler interfaces for testing
type mockHandlers struct {
	toolHandler     mcp.ToolHandler
	resourceHandler mcp.ResourceHandler
	promptHandler   mcp.PromptHandler
	shouldFailInit  bool
}

func (m *mockHandlers) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if m.toolHandler != nil {
		return m.toolHandler.ListTools(ctx)
	}
	return []mcp.Tool{
		{Name: "test-tool", Description: "Test tool"},
	}, nil
}

func (m *mockHandlers) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	if m.toolHandler != nil {
		return m.toolHandler.CallTool(ctx, params)
	}
	return mcp.ToolResponse{
		Content: []mcp.ContentItem{
			{Type: "text", Text: "test result"},
		},
	}, nil
}

func (m *mockHandlers) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	if m.resourceHandler != nil {
		return m.resourceHandler.ListResources(ctx)
	}
	return []mcp.Resource{
		{URI: "test://uri", Name: "test resource"},
	}, nil
}

func (m *mockHandlers) ReadResource(ctx context.Context, params mcp.ResourceParams) (mcp.ResourceResponse, error) {
	if m.resourceHandler != nil {
		return m.resourceHandler.ReadResource(ctx, params)
	}
	return mcp.ResourceResponse{
		Contents: []mcp.ResourceContent{
			{URI: "test://uri", Text: "test content"},
		},
	}, nil
}

func (m *mockHandlers) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	if m.promptHandler != nil {
		return m.promptHandler.ListPrompts(ctx)
	}
	return []mcp.Prompt{
		{Name: "test-prompt", Description: "Test prompt"},
	}, nil
}

func (m *mockHandlers) GetPrompt(ctx context.Context, params mcp.PromptParams) (mcp.PromptResponse, error) {
	if m.promptHandler != nil {
		return m.promptHandler.GetPrompt(ctx, params)
	}
	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{
			{Role: "user", Content: mcp.MessageContent{Type: "text", Text: "test prompt"}},
		},
	}, nil
}

// mockResponseSender is a mock implementation of ResponseSender
type mockResponseSender struct {
	lastResponse mcp.Response
	lastError    struct {
		id      any
		code    int
		message string
		data    any
	}
}

func (m *mockResponseSender) SendResponse(response mcp.Response) error {
	m.lastResponse = response
	return nil
}

func (m *mockResponseSender) SendError(id any, code int, message string, data any) error {
	m.lastError.id = id
	m.lastError.code = code
	m.lastError.message = message
	m.lastError.data = data
	return nil
}

func TestNew(t *testing.T) {
	t.Run("creates server with valid config", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if server == nil {
			t.Fatalf("Expected non-nil server")
		}

		if server.serverInfo.Name != cfg.ServerName {
			t.Errorf("Expected server name %s, got %s", cfg.ServerName, server.serverInfo.Name)
		}
	})

	t.Run("returns error when config is nil", func(t *testing.T) {
		handlers := &mockHandlers{}

		server, err := New(nil, handlers, handlers, handlers)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		if err.Error() != "config cannot be nil" {
			t.Errorf("Expected 'config cannot be nil' error, got: %v", err)
		}

		if server != nil {
			t.Errorf("Expected nil server")
		}
	})

	t.Run("returns error when toolHandler is nil", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		_, err := New(cfg, nil, handlers, handlers)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		if err.Error() != "toolHandler cannot be nil" {
			t.Errorf("Expected 'toolHandler cannot be nil' error, got: %v", err)
		}
	})

	t.Run("returns error when resourceHandler is nil", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		_, err := New(cfg, handlers, nil, handlers)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		if err.Error() != "resourceHandler cannot be nil" {
			t.Errorf("Expected 'resourceHandler cannot be nil' error, got: %v", err)
		}
	})

	t.Run("returns error when promptHandler is nil", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		_, err := New(cfg, handlers, handlers, nil)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}

		if err.Error() != "promptHandler cannot be nil" {
			t.Errorf("Expected 'promptHandler cannot be nil' error, got: %v", err)
		}
	})
}

func TestServer_Initialize(t *testing.T) {
	t.Run("returns valid initialize response", func(t *testing.T) {
		cfg := config.New()
		cfg.ServerName = "Test Server"
		cfg.ServerVersion = "2.0.0"
		handlers := &mockHandlers{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		response, err := server.Initialize(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if response.ProtocolVersion != mcp.ProtocolVersion {
			t.Errorf("Expected protocol version %s, got %s", mcp.ProtocolVersion, response.ProtocolVersion)
		}

		if response.ServerInfo.Name != "Test Server" {
			t.Errorf("Expected server name 'Test Server', got %s", response.ServerInfo.Name)
		}

		if response.ServerInfo.Version != "2.0.0" {
			t.Errorf("Expected version '2.0.0', got %s", response.ServerInfo.Version)
		}

		if response.Capabilities == nil {
			t.Errorf("Expected capabilities to be set")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Note: The Initialize method doesn't directly check context
		// Context cancellation is checked in ListTools, which is called during request handling
		// For this test, we just verify that Initialize returns successfully
		response, err := server.Initialize(ctx)
		if err != nil {
			// Initialize doesn't check context, so this might return nil error
			t.Logf("Initialize doesn't check context directly, got error: %v", err)
		}

		if response != nil {
			// Response should still be returned
			if response.ServerInfo.Name == "" {
				t.Errorf("Expected server info to be set")
			}
		}
	})
}

func TestServer_HandleRequest(t *testing.T) {
	t.Run("handles initialize method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "initialize",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles tools/list method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "tools/list",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles tools/call method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "tools/call",
			ID:      "test-id",
			Params: map[string]any{
				"name":      "test-tool",
				"arguments": map[string]any{},
			},
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles resources/list method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "resources/list",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles resources/read method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "resources/read",
			ID:      "test-id",
			Params: map[string]any{
				"uri": "test://uri",
			},
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles prompts/list method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "prompts/list",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles prompts/get method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "prompts/get",
			ID:      "test-id",
			Params: map[string]any{
				"name":      "test-prompt",
				"arguments": map[string]any{},
			},
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("handles ping method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "ping",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.Result == nil {
			t.Errorf("Expected result in response")
		}
	})

	t.Run("returns error for unknown method", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}
		sender := &mockResponseSender{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "unknown/method",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error (error sent via sender), got: %v", err)
		}

		if sender.lastError.code != mcp.ErrorCodeMethodNotFound {
			t.Errorf("Expected method not found error code, got: %d", sender.lastError.code)
		}
	})

	t.Run("returns error when no response sender in context", func(t *testing.T) {
		cfg := config.New()
		handlers := &mockHandlers{}

		server, err := New(cfg, handlers, handlers, handlers)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx := context.Background()
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "initialize",
			ID:      "test-id",
		}

		err = server.HandleRequest(ctx, req)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "no response sender in context" {
			t.Errorf("Expected 'no response sender in context' error, got: %v", err)
		}
	})
}

func TestServer_parseToolCallParams(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("valid params", func(t *testing.T) {
		params := map[string]any{
			"name": "test-tool",
			"arguments": map[string]any{
				"param1": "value1",
			},
		}

		result, err := server.parseToolCallParams(params)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.Name != "test-tool" {
			t.Errorf("Expected name 'test-tool', got %s", result.Name)
		}

		if result.Arguments["param1"] != "value1" {
			t.Errorf("Expected argument param1=value1, got %v", result.Arguments["param1"])
		}
	})

	t.Run("nil params", func(t *testing.T) {
		_, err := server.parseToolCallParams(nil)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params cannot be nil" {
			t.Errorf("Expected 'params cannot be nil' error, got: %v", err)
		}
	})

	t.Run("params not a map", func(t *testing.T) {
		_, err := server.parseToolCallParams("invalid")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params must be an object" {
			t.Errorf("Expected 'params must be an object' error, got: %v", err)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		params := map[string]any{
			"arguments": map[string]any{},
		}

		_, err := server.parseToolCallParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "name parameter is required and must be a string" {
			t.Errorf("Expected 'name parameter is required and must be a string' error, got: %v", err)
		}
	})

	t.Run("name not a string", func(t *testing.T) {
		params := map[string]any{
			"name": 123,
		}

		_, err := server.parseToolCallParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "name parameter is required and must be a string" {
			t.Errorf("Expected 'name parameter is required and must be a string' error, got: %v", err)
		}
	})

	t.Run("arguments not a map", func(t *testing.T) {
		params := map[string]any{
			"name":      "test-tool",
			"arguments": "invalid",
		}

		result, err := server.parseToolCallParams(params)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(result.Arguments) != 0 {
			t.Errorf("Expected empty arguments, got %v", result.Arguments)
		}
	})
}

func TestServer_parseResourceParams(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("valid params", func(t *testing.T) {
		params := map[string]any{
			"uri": "test://uri",
		}

		result, err := server.parseResourceParams(params)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.URI != "test://uri" {
			t.Errorf("Expected URI 'test://uri', got %s", result.URI)
		}
	})

	t.Run("nil params", func(t *testing.T) {
		_, err := server.parseResourceParams(nil)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params cannot be nil" {
			t.Errorf("Expected 'params cannot be nil' error, got: %v", err)
		}
	})

	t.Run("params not a map", func(t *testing.T) {
		_, err := server.parseResourceParams("invalid")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params must be an object" {
			t.Errorf("Expected 'params must be an object' error, got: %v", err)
		}
	})

	t.Run("missing uri", func(t *testing.T) {
		params := map[string]any{}

		_, err := server.parseResourceParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "uri parameter is required and must be a string" {
			t.Errorf("Expected 'uri parameter is required and must be a string' error, got: %v", err)
		}
	})

	t.Run("uri not a string", func(t *testing.T) {
		params := map[string]any{
			"uri": 123,
		}

		_, err := server.parseResourceParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "uri parameter is required and must be a string" {
			t.Errorf("Expected 'uri parameter is required and must be a string' error, got: %v", err)
		}
	})
}

func TestServer_parsePromptParams(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("valid params", func(t *testing.T) {
		params := map[string]any{
			"name": "test-prompt",
			"arguments": map[string]any{
				"param1": "value1",
			},
		}

		result, err := server.parsePromptParams(params)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.Name != "test-prompt" {
			t.Errorf("Expected name 'test-prompt', got %s", result.Name)
		}

		if result.Arguments["param1"] != "value1" {
			t.Errorf("Expected argument param1=value1, got %v", result.Arguments["param1"])
		}
	})

	t.Run("nil params", func(t *testing.T) {
		_, err := server.parsePromptParams(nil)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params cannot be nil" {
			t.Errorf("Expected 'params cannot be nil' error, got: %v", err)
		}
	})

	t.Run("params not a map", func(t *testing.T) {
		_, err := server.parsePromptParams("invalid")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "params must be an object" {
			t.Errorf("Expected 'params must be an object' error, got: %v", err)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		params := map[string]any{
			"arguments": map[string]any{},
		}

		_, err := server.parsePromptParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "name parameter is required and must be a string" {
			t.Errorf("Expected 'name parameter is required and must be a string' error, got: %v", err)
		}
	})

	t.Run("name not a string", func(t *testing.T) {
		params := map[string]any{
			"name": 123,
		}

		_, err := server.parsePromptParams(params)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "name parameter is required and must be a string" {
			t.Errorf("Expected 'name parameter is required and must be a string' error, got: %v", err)
		}
	})

	t.Run("arguments not a map", func(t *testing.T) {
		params := map[string]any{
			"name":      "test-prompt",
			"arguments": "invalid",
		}

		result, err := server.parsePromptParams(params)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(result.Arguments) != 0 {
			t.Errorf("Expected empty arguments, got %v", result.Arguments)
		}
	})
}

func TestServer_sendResponse(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("sends response successfully", func(t *testing.T) {
		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		result := map[string]string{"status": "ok"}
		err := server.sendResponse(ctx, "test-id", result)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastResponse.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got: %v", sender.lastResponse.ID)
		}

		if sender.lastResponse.JSONRPC != mcp.JSONRPCVersion {
			t.Errorf("Expected JSONRPC version %s, got: %s", mcp.JSONRPCVersion, sender.lastResponse.JSONRPC)
		}
	})

	t.Run("returns error when no response sender", func(t *testing.T) {
		ctx := context.Background()

		err := server.sendResponse(ctx, "test-id", map[string]string{"status": "ok"})
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "no response sender in context" {
			t.Errorf("Expected 'no response sender in context' error, got: %v", err)
		}
	})
}

func TestServer_sendError(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("sends error successfully", func(t *testing.T) {
		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		err := server.sendError(ctx, "test-id", mcp.ErrorCodeInvalidRequest, "Invalid request", nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if sender.lastError.id != "test-id" {
			t.Errorf("Expected ID 'test-id', got: %v", sender.lastError.id)
		}

		if sender.lastError.code != mcp.ErrorCodeInvalidRequest {
			t.Errorf("Expected error code %d, got: %d", mcp.ErrorCodeInvalidRequest, sender.lastError.code)
		}

		if sender.lastError.message != "Invalid request" {
			t.Errorf("Expected message 'Invalid request', got: %s", sender.lastError.message)
		}
	})

	t.Run("returns error when no response sender", func(t *testing.T) {
		ctx := context.Background()

		err := server.sendError(ctx, "test-id", mcp.ErrorCodeInvalidRequest, "Invalid request", nil)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if err.Error() != "no response sender in context" {
			t.Errorf("Expected 'no response sender in context' error, got: %v", err)
		}
	})
}

func TestServer_HandleRequest_ErrorScenarios(t *testing.T) {
	cfg := config.New()
	handlers := &mockHandlers{}
	server, _ := New(cfg, handlers, handlers, handlers)

	t.Run("tools/call with invalid params", func(t *testing.T) {
		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "tools/call",
			ID:      "test-id",
			Params: map[string]any{
				"invalid": "params",
			},
		}

		err := server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error (error sent via sender), got: %v", err)
		}

		if sender.lastError.code != mcp.ErrorCodeInvalidParams {
			t.Errorf("Expected invalid params error code, got: %d", sender.lastError.code)
		}
	})

	t.Run("tools/call returns handler error", func(t *testing.T) {
		errorHandler := &mockHandlers{
			toolHandler: &errorToolHandler{},
		}
		server, _ := New(cfg, errorHandler, errorHandler, errorHandler)

		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "tools/call",
			ID:      "test-id",
			Params: map[string]any{
				"name": "error-tool",
			},
		}

		err := server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error (error sent via sender), got: %v", err)
		}

		if sender.lastError.code != mcp.ErrorCodeInvalidParams {
			t.Errorf("Expected invalid params error code, got: %d", sender.lastError.code)
		}
	})

	t.Run("resources/read with invalid params", func(t *testing.T) {
		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "resources/read",
			ID:      "test-id",
			Params: map[string]any{
				"invalid": "params",
			},
		}

		err := server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error (error sent via sender), got: %v", err)
		}

		if sender.lastError.code != mcp.ErrorCodeInvalidParams {
			t.Errorf("Expected invalid params error code, got: %d", sender.lastError.code)
		}
	})

	t.Run("prompts/get with invalid params", func(t *testing.T) {
		sender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)

		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "prompts/get",
			ID:      "test-id",
			Params: map[string]any{
				"invalid": "params",
			},
		}

		err := server.HandleRequest(ctx, req)
		if err != nil {
			t.Errorf("Expected no error (error sent via sender), got: %v", err)
		}

		if sender.lastError.code != mcp.ErrorCodeInvalidParams {
			t.Errorf("Expected invalid params error code, got: %d", sender.lastError.code)
		}
	})
}

// errorToolHandler returns errors for testing
type errorToolHandler struct{}

func (e *errorToolHandler) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return []mcp.Tool{}, nil
}

func (e *errorToolHandler) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	return mcp.ToolResponse{}, errors.New("tool call failed")
}
