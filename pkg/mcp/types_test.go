package mcp

import (
	"context"
	"testing"
)

func TestConstants(t *testing.T) {
	t.Run("protocol version", func(t *testing.T) {
		if ProtocolVersion != "2025-03-26" {
			t.Errorf("Expected ProtocolVersion '2025-03-26', got %s", ProtocolVersion)
		}
	})

	t.Run("JSON-RPC version", func(t *testing.T) {
		if JSONRPCVersion != "2.0" {
			t.Errorf("Expected JSONRPCVersion '2.0', got %s", JSONRPCVersion)
		}
	})

	t.Run("error codes", func(t *testing.T) {
		if ErrorCodeParseError != -32700 {
			t.Errorf("Expected ErrorCodeParseError -32700, got %d", ErrorCodeParseError)
		}

		if ErrorCodeInvalidRequest != -32600 {
			t.Errorf("Expected ErrorCodeInvalidRequest -32600, got %d", ErrorCodeInvalidRequest)
		}

		if ErrorCodeMethodNotFound != -32601 {
			t.Errorf("Expected ErrorCodeMethodNotFound -32601, got %d", ErrorCodeMethodNotFound)
		}

		if ErrorCodeInvalidParams != -32602 {
			t.Errorf("Expected ErrorCodeInvalidParams -32602, got %d", ErrorCodeInvalidParams)
		}

		if ErrorCodeInternalError != -32603 {
			t.Errorf("Expected ErrorCodeInternalError -32603, got %d", ErrorCodeInternalError)
		}
	})

	t.Run("context keys", func(t *testing.T) {
		if ResponseSenderKey != "responseSender" {
			t.Errorf("Expected ResponseSenderKey 'responseSender', got %s", ResponseSenderKey)
		}

		if SessionIDKey != "sessionID" {
			t.Errorf("Expected SessionIDKey 'sessionID', got %s", SessionIDKey)
		}
	})
}

func TestRequest(t *testing.T) {
	t.Run("create request with ID", func(t *testing.T) {
		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "test.method",
			ID:      123,
			Params:  map[string]any{"param": "value"},
		}

		if req.JSONRPC != JSONRPCVersion {
			t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, req.JSONRPC)
		}

		if req.Method != "test.method" {
			t.Errorf("Expected method 'test.method', got %s", req.Method)
		}

		if req.ID != 123 {
			t.Errorf("Expected ID 123, got %v", req.ID)
		}
	})

	t.Run("create request with nil ID (notification)", func(t *testing.T) {
		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "test.notification",
			ID:      nil,
		}

		if req.ID != nil {
			t.Errorf("Expected nil ID for notification, got %v", req.ID)
		}
	})

	t.Run("create request with string ID", func(t *testing.T) {
		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "test.method",
			ID:      "test-id",
		}

		if req.ID != "test-id" {
			t.Errorf("Expected string ID 'test-id', got %v", req.ID)
		}
	})

	t.Run("create request without params", func(t *testing.T) {
		req := Request{
			JSONRPC: JSONRPCVersion,
			Method:  "test.method",
			ID:      1,
		}

		if req.Params != nil {
			t.Errorf("Expected nil params, got %v", req.Params)
		}
	})
}

func TestResponse(t *testing.T) {
	t.Run("create success response", func(t *testing.T) {
		resp := Response{
			JSONRPC: JSONRPCVersion,
			ID:      "test-id",
			Result:  map[string]string{"status": "ok"},
		}

		if resp.JSONRPC != JSONRPCVersion {
			t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, resp.JSONRPC)
		}

		if resp.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got %v", resp.ID)
		}

		if resp.Result == nil {
			t.Errorf("Expected result to be set")
		}

		if resp.Error != nil {
			t.Errorf("Expected error to be nil for success response")
		}
	})

	t.Run("create error response", func(t *testing.T) {
		errorResp := &ErrorResponse{
			Code:    ErrorCodeInvalidRequest,
			Message: "Invalid request",
			Data:    "Additional error data",
		}

		resp := Response{
			JSONRPC: JSONRPCVersion,
			ID:      "test-id",
			Error:   errorResp,
		}

		if resp.Error == nil {
			t.Errorf("Expected error to be set")
		}

		if resp.Error.Code != ErrorCodeInvalidRequest {
			t.Errorf("Expected error code %d, got %d", ErrorCodeInvalidRequest, resp.Error.Code)
		}

		if resp.Error.Message != "Invalid request" {
			t.Errorf("Expected error message 'Invalid request', got %s", resp.Error.Message)
		}

		if resp.Error.Data != "Additional error data" {
			t.Errorf("Expected error data 'Additional error data', got %v", resp.Error.Data)
		}

		if resp.Result != nil {
			t.Errorf("Expected result to be nil for error response")
		}
	})

	t.Run("create response with number ID", func(t *testing.T) {
		resp := Response{
			JSONRPC: JSONRPCVersion,
			ID:      456,
			Result:  nil,
		}

		if resp.ID != 456 {
			t.Errorf("Expected number ID 456, got %v", resp.ID)
		}
	})
}

func TestErrorResponse(t *testing.T) {
	t.Run("create error with minimal fields", func(t *testing.T) {
		errResp := ErrorResponse{
			Code:    ErrorCodeInternalError,
			Message: "Internal error",
		}

		if errResp.Code != ErrorCodeInternalError {
			t.Errorf("Expected error code %d, got %d", ErrorCodeInternalError, errResp.Code)
		}

		if errResp.Message != "Internal error" {
			t.Errorf("Expected message 'Internal error', got %s", errResp.Message)
		}

		if errResp.Data != nil {
			t.Errorf("Expected nil data")
		}
	})

	t.Run("create error with all fields", func(t *testing.T) {
		data := map[string]any{"field": "value"}
		errResp := ErrorResponse{
			Code:    ErrorCodeInvalidParams,
			Message: "Invalid parameters",
			Data:    data,
		}

		if errResp.Data == nil {
			t.Errorf("Expected data to be set")
		}

		if d, ok := errResp.Data.(map[string]any); ok {
			if d["field"] != "value" {
				t.Errorf("Expected data field 'value', got %v", d["field"])
			}
		} else {
			t.Errorf("Expected data to be map[string]any")
		}
	})
}

func TestServerInfo(t *testing.T) {
	t.Run("create server info", func(t *testing.T) {
		info := ServerInfo{
			Name:    "Test Server",
			Version: "1.0.0",
		}

		if info.Name != "Test Server" {
			t.Errorf("Expected name 'Test Server', got %s", info.Name)
		}

		if info.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got %s", info.Version)
		}
	})
}

func TestInitializeResponse(t *testing.T) {
	t.Run("create initialize response", func(t *testing.T) {
		resp := InitializeResponse{
			ProtocolVersion: ProtocolVersion,
			Capabilities: map[string]any{
				"tools": map[string]bool{"listChanged": true},
			},
			ServerInfo: ServerInfo{
				Name:    "Test Server",
				Version: "1.0.0",
			},
		}

		if resp.ProtocolVersion != ProtocolVersion {
			t.Errorf("Expected protocol version %s, got %s", ProtocolVersion, resp.ProtocolVersion)
		}

		if resp.Capabilities == nil {
			t.Errorf("Expected capabilities to be set")
		}

		if resp.ServerInfo.Name != "Test Server" {
			t.Errorf("Expected server name 'Test Server', got %s", resp.ServerInfo.Name)
		}
	})
}

func TestContextKeys(t *testing.T) {
	t.Run("store and retrieve response sender from context", func(t *testing.T) {
		mockSender := &mockResponseSender{}
		ctx := context.WithValue(context.Background(), ResponseSenderKey, mockSender)

		retrieved := ctx.Value(ResponseSenderKey)
		if retrieved == nil {
			t.Errorf("Expected response sender in context")
		}

		if retrieved != mockSender {
			t.Errorf("Retrieved value doesn't match stored value")
		}
	})

	t.Run("store and retrieve session ID from context", func(t *testing.T) {
		sessionID := "test-session-123"
		ctx := context.WithValue(context.Background(), SessionIDKey, sessionID)

		retrieved := ctx.Value(SessionIDKey)
		if retrieved == nil {
			t.Errorf("Expected session ID in context")
		}

		if retrieved != sessionID {
			t.Errorf("Expected session ID %s, got %v", sessionID, retrieved)
		}
	})

	t.Run("get nil for missing context value", func(t *testing.T) {
		ctx := context.Background()

		retrieved := ctx.Value(ResponseSenderKey)
		if retrieved != nil {
			t.Errorf("Expected nil for missing context value, got %v", retrieved)
		}
	})
}

// mockResponseSender is a minimal mock for testing
type mockResponseSender struct{}

func (m *mockResponseSender) SendResponse(response Response) error {
	return nil
}

func (m *mockResponseSender) SendError(id any, code int, message string, data any) error {
	return nil
}
