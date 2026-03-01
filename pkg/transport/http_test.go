package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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
	req.Header.Set("Content-Type", "application/json")

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
	req.Header.Set("Content-Type", "application/json")

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
	req.Header.Set("Content-Type", "application/json")

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
	initHTTPReq.Header.Set("Content-Type", "application/json")
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(mcp.SessionIDHeader, sessionID)
	req.Header.Set(mcp.ProtocolVersionHeader, "2099-01-01")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRequiresContentType(t *testing.T) {
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

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status 415, got %d", rr.Code)
	}
}

func TestHandlePostRequiresBothAcceptTypes(t *testing.T) {
	tx := newHTTPTransportForTest()
	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusNotAcceptable {
		t.Fatalf("expected status 406, got %d", rr.Code)
	}
}

func TestHandlePostRejectsMultipleMessages(t *testing.T) {
	tx := newHTTPTransportForTest()
	body := []byte(`{"jsonrpc":"2.0","method":"initialize","id":1}{"jsonrpc":"2.0","method":"initialize","id":2}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRejectsInitializeWithSessionHeader(t *testing.T) {
	tx := newHTTPTransportForTest()
	tx.registerSession("existing-session")
	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, "existing-session")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostAcceptParsingSupportsQValues(t *testing.T) {
	tx := newHTTPTransportForTest()
	reqBody := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;q=1.0, text/event-stream;q=0.9")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestIsOriginAllowedDoesNotMatchSubstringHosts(t *testing.T) {
	tx := newHTTPTransportForTest()

	if tx.isOriginAllowed("http://notlocalhost.evil.com") {
		t.Fatal("expected disallowed origin with localhost substring")
	}

	if !tx.isOriginAllowed("http://localhost:9000") {
		t.Fatal("expected localhost origin to be allowed")
	}
}

func TestOriginPatternEscapesRegexMetacharacters(t *testing.T) {
	cfg := &config.Config{
		TransportType:   "http",
		HTTPPort:        8080,
		RequestTimeout:  time.Second,
		ShutdownTimeout: time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		AllowedOrigins:  []string{"https://api.example.com"},
	}

	tx := NewHTTP(cfg)

	if !tx.isOriginAllowed("https://api.example.com") {
		t.Fatal("expected exact configured origin to be allowed")
	}

	if tx.isOriginAllowed("https://apiXexampleYcom") {
		t.Fatal("expected origin with regex-like substitutions to be disallowed")
	}
}

func TestHandleDeleteCleansEventCounter(t *testing.T) {
	tx := newHTTPTransportForTest()
	sessionID := "session-1"
	tx.registerSession(sessionID)
	tx.setEventCounter(sessionID, 10)

	req := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	req.Header.Set(mcp.SessionIDHeader, sessionID)
	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)

	rr := httptest.NewRecorder()
	tx.handleDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}

	tx.mu.RLock()
	_, exists := tx.eventCounters[sessionID]
	tx.mu.RUnlock()
	if exists {
		t.Fatal("expected event counter to be removed when session is deleted")
	}
}

func TestStartReturnsListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	cfg := &config.Config{
		TransportType:   "http",
		HTTPPort:        port,
		RequestTimeout:  time.Second,
		ShutdownTimeout: time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		AllowedOrigins:  []string{"http://localhost:*"},
	}

	tx := NewHTTP(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = tx.Start(ctx, &httpMockServer{})
	if err == nil {
		t.Fatal("expected startup error when port is already bound")
	}
	if !strings.Contains(err.Error(), "http server failed") {
		t.Fatalf("expected startup failure error, got %v", err)
	}
}

func TestEnsureProtocolVersion(t *testing.T) {
	tx := newHTTPTransportForTest()

	if err := tx.ensureProtocolVersion(""); err == nil {
		t.Fatal("expected missing protocol version to fail")
	}
	if err := tx.ensureProtocolVersion("2099-01-01"); err == nil {
		t.Fatal("expected unsupported protocol version to fail")
	}
	if err := tx.ensureProtocolVersion(mcp.ProtocolVersion); err != nil {
		t.Fatalf("expected current protocol version to pass: %v", err)
	}
	if err := tx.ensureProtocolVersion(mcp.LegacyProtocolVersion); err != nil {
		t.Fatalf("expected legacy protocol version to pass: %v", err)
	}
}

func TestValidateExistingSession(t *testing.T) {
	tx := newHTTPTransportForTest()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)

	if err := tx.validateExistingSession(req); err == nil {
		t.Fatal("expected validation error when headers are missing")
	}

	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)
	req.Header.Set(mcp.SessionIDHeader, "missing-session")
	if err := tx.validateExistingSession(req); err == nil {
		t.Fatal("expected unknown session validation error")
	}

	tx.registerSession("known-session")
	req.Header.Set(mcp.SessionIDHeader, "known-session")
	if err := tx.validateExistingSession(req); err != nil {
		t.Fatalf("expected known session validation success: %v", err)
	}
}

func TestParseLastEventID(t *testing.T) {
	streamID, seq, ok := parseLastEventID("abc:123")
	if !ok || streamID != "abc" || seq != 123 {
		t.Fatalf("expected valid event id parse, got (%s, %d, %v)", streamID, seq, ok)
	}

	if _, _, ok := parseLastEventID(""); ok {
		t.Fatal("expected empty event id to fail")
	}
	if _, _, ok := parseLastEventID("abc"); ok {
		t.Fatal("expected event id missing separator to fail")
	}
	if _, _, ok := parseLastEventID(":123"); ok {
		t.Fatal("expected empty stream id to fail")
	}
	if _, _, ok := parseLastEventID("abc:notanumber"); ok {
		t.Fatal("expected non-numeric sequence to fail")
	}
}
