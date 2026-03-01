package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

type httpMockServer struct{}

func (m *httpMockServer) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return &mcp.InitializeResponse{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    map[string]any{},
		ServerInfo: mcp.ServerInfo{
			Name:    "test",
			Version: "1.0.0",
		},
	}, nil
}

func (m *httpMockServer) HandleRequest(ctx context.Context, req mcp.Request) error {
	sender := ctx.Value(mcp.ResponseSenderKey).(mcp.ResponseSender)
	return sender.SendResponse(mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"ok": true},
	})
}

func newHTTPTransportForTest() *HTTPTransport {
	cfg := &config.Config{
		TransportType:   "http",
		HTTPPort:        8080,
		RequestTimeout:  time.Second,
		ShutdownTimeout: time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		AllowedOrigins:  []string{"http://localhost:*"},
	}
	return NewHTTP(cfg)
}

func TestHandlePostInitializeSetsSessionHeader(t *testing.T) {
	tx := newHTTPTransportForTest()
	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Header().Get(mcp.SessionIDHeader) == "" {
		t.Fatal("expected MCP session header to be set")
	}
}

func TestHandlePostRejectsBatchRequests(t *testing.T) {
	tx := newHTTPTransportForTest()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`[{"jsonrpc":"2.0","method":"initialize","id":1}]`)))
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRequiresSessionAfterInitialize(t *testing.T) {
	tx := newHTTPTransportForTest()
	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRejectsUnsupportedProtocolHeader(t *testing.T) {
	tx := newHTTPTransportForTest()
	initReq := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1}
	initBytes, err := json.Marshal(initReq)
	if err != nil {
		t.Fatalf("marshal initialize request: %v", err)
	}

	initHTTPReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(initBytes))
	initHTTPReq.Header.Set("Accept", "application/json, text/event-stream")
	initRecorder := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, initRecorder, initHTTPReq)
	sessionID := initRecorder.Header().Get(mcp.SessionIDHeader)
	if sessionID == "" {
		t.Fatal("expected session id from initialize")
	}

	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 2}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, sessionID)
	req.Header.Set(mcp.ProtocolVersionHeader, "2099-01-01")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
