package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type httpMockServerNoResponse struct{}

type httpMockServerError struct{}

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
	sender, ok := ctx.Value(mcp.ResponseSenderKey).(mcp.ResponseSender)
	if !ok {
		return fmt.Errorf("response sender missing or wrong type")
	}
	return sender.SendResponse(mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"ok": true},
	})
}

func (m *httpMockServerNoResponse) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return (&httpMockServer{}).Initialize(ctx)
}

func (m *httpMockServerNoResponse) HandleRequest(ctx context.Context, req mcp.Request) error {
	return nil
}

func (m *httpMockServerError) Initialize(ctx context.Context) (*mcp.InitializeResponse, error) {
	return (&httpMockServer{}).Initialize(ctx)
}

func (m *httpMockServerError) HandleRequest(ctx context.Context, req mcp.Request) error {
	return errors.New("boom")
}

type nonFlusherResponseWriter struct {
	headers http.Header
	status  int
	body    bytes.Buffer
}

func (w *nonFlusherResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *nonFlusherResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *nonFlusherResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func newHTTPTransportForTest(overrides ...func(*config.Config)) *HTTPTransport {
	cfg := config.New()
	cfg.TransportType = "http"
	cfg.HTTPPort = 8080
	cfg.RequestTimeout = time.Second
	cfg.ShutdownTimeout = time.Second
	cfg.ReadTimeout = time.Second
	cfg.WriteTimeout = time.Second
	cfg.IdleTimeout = time.Second
	cfg.AllowedOrigins = []string{"http://localhost:*"}

	for _, override := range overrides {
		override(cfg)
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

func TestHandlePostRejectsMalformedJSON(t *testing.T) {
	tx := newHTTPTransportForTest()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRejectsEmptyBody(t *testing.T) {
	tx := newHTTPTransportForTest()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("   \n\t  ")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRejectsWrongJSONRPCVersion(t *testing.T) {
	tx := newHTTPTransportForTest()
	reqBody := mcp.Request{JSONRPC: "1.0", Method: "initialize", ID: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostRejectsInitializeNotification(t *testing.T) {
	tx := newHTTPTransportForTest()
	body := []byte(`{"jsonrpc":"2.0","method":"initialize"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestHandlePostNotificationAccepted(t *testing.T) {
	tx := newHTTPTransportForTest()
	tx.registerSession("session-1")
	body := []byte(`{"jsonrpc":"2.0","method":"tools/list"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, "session-1")
	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
}

func TestHandlePostResponseAcceptedForKnownSession(t *testing.T) {
	tx := newHTTPTransportForTest()
	tx.registerSession("session-1")
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, "session-1")
	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)

	rr := httptest.NewRecorder()
	tx.handlePost(context.Background(), &httpMockServer{}, rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
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
	tx := newHTTPTransportForTest(func(cfg *config.Config) {
		cfg.AllowedOrigins = []string{"https://api.example.com"}
	})

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
	tx := newHTTPTransportForTest(func(cfg *config.Config) {
		cfg.HTTPPort = port
	})
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

func TestHTTPResponseSender(t *testing.T) {
	rr := httptest.NewRecorder()
	s := &HTTPResponseSender{writer: rr, sessionID: "session-1"}

	err := s.SendResponse(mcp.Response{JSONRPC: mcp.JSONRPCVersion, ID: 1, Result: map[string]any{"ok": true}})
	if err != nil {
		t.Fatalf("SendResponse failed: %v", err)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get(mcp.SessionIDHeader) != "session-1" {
		t.Fatalf("expected session header to be set")
	}

	err = s.SendResponse(mcp.Response{JSONRPC: mcp.JSONRPCVersion, ID: 2})
	if err == nil {
		t.Fatal("expected error on second response send")
	}
}

func TestHTTPResponseSenderSendError(t *testing.T) {
	rr := httptest.NewRecorder()
	s := &HTTPResponseSender{writer: rr}

	err := s.SendError(1, mcp.ErrorCodeInternalError, "oops", "data")
	if err != nil {
		t.Fatalf("SendError failed: %v", err)
	}
	if !strings.Contains(rr.Body.String(), "oops") {
		t.Fatalf("expected body to contain error message, got %s", rr.Body.String())
	}
}

func TestSSESessionSendEventAndError(t *testing.T) {
	rr := httptest.NewRecorder()
	session := &SSESession{
		ID:      "session-1",
		writer:  rr,
		flusher: rr,
		nextEventID: func() string {
			return "session-1:1"
		},
	}

	err := session.sendEvent("connected", map[string]any{"a": "b"})
	if err != nil {
		t.Fatalf("sendEvent failed: %v", err)
	}

	err = session.sendError(1, mcp.ErrorCodeInternalError, "boom", nil)
	if err != nil {
		t.Fatalf("sendError failed: %v", err)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "event: connected") || !strings.Contains(body, `"code":-32603`) {
		t.Fatalf("unexpected SSE body: %s", body)
	}

	session.close()
	err = session.sendEvent("connected", map[string]any{"a": "b"})
	if err == nil {
		t.Fatal("expected error when sending on closed session")
	}
}

func TestStartSSEStreamBranches(t *testing.T) {
	tx := newHTTPTransportForTest()
	tx.registerSession("session-1")

	nonFlusher := &nonFlusherResponseWriter{}
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	if session := tx.startSSEStream(nonFlusher, req, "session-1"); session != nil {
		t.Fatal("expected nil session when writer is not a flusher")
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Last-Event-ID", "bad-value")
	rr := httptest.NewRecorder()
	if session := tx.startSSEStream(rr, req, "session-1"); session != nil {
		t.Fatal("expected nil session for invalid last-event-id")
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr = httptest.NewRecorder()
	if session := tx.startSSEStream(rr, req, "unknown"); session != nil {
		t.Fatal("expected nil session for unknown session")
	}
}

func TestHandleGetGuardsAndCleanup(t *testing.T) {
	tx := newHTTPTransportForTest()
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr := httptest.NewRecorder()
	tx.handleGet(ctx, &httpMockServer{}, rr, req)
	if rr.Code != http.StatusNotAcceptable {
		t.Fatalf("expected 406, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	rr = httptest.NewRecorder()
	tx.handleGet(ctx, &httpMockServer{}, rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing session header, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, "session-1")
	rr = httptest.NewRecorder()
	tx.handleGet(ctx, &httpMockServer{}, rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing protocol header, got %d", rr.Code)
	}

	tx.registerSession("session-1")
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set(mcp.SessionIDHeader, "session-1")
	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)
	rr = httptest.NewRecorder()
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	tx.handleGet(cancelledCtx, &httpMockServer{}, rr, req)

	tx.mu.RLock()
	_, exists := tx.sessions["session-1"]
	tx.mu.RUnlock()
	if exists {
		t.Fatal("expected session to be cleaned up after context cancellation")
	}
}

func TestHandleDeleteGuards(t *testing.T) {
	tx := newHTTPTransportForTest()

	req := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	rr := httptest.NewRecorder()
	tx.handleDelete(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing session header, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	req.Header.Set(mcp.SessionIDHeader, "unknown")
	req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)
	rr = httptest.NewRecorder()
	tx.handleDelete(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown session, got %d", rr.Code)
	}
}

func TestHandleJSONRequestBranches(t *testing.T) {
	tx := newHTTPTransportForTest()
	req := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 1}

	rr := httptest.NewRecorder()
	tx.handleJSONRequest(context.Background(), &httpMockServerError{}, rr, req, "")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on server error, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	tx.handleJSONRequest(context.Background(), &httpMockServerNoResponse{}, rr, req, "")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when no response is generated, got %d", rr.Code)
	}
}

func TestStopAndMiddleware(t *testing.T) {
	tx := newHTTPTransportForTest()
	if err := tx.Stop(); err != nil {
		t.Fatalf("expected stop without server to succeed, got %v", err)
	}

	h := tx.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "http://localhost:1234")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS header to be set for allowed origin")
	}

	req = httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected no CORS origin header for missing Origin request header")
	}

	secured := tx.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Origin", "http://evil.com")
	rr = httptest.NewRecorder()
	secured.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", rr.Code)
	}
}

func TestHTTPTransportSendError(t *testing.T) {
	tx := newHTTPTransportForTest()
	rr := httptest.NewRecorder()
	tx.sendError(rr, "id-1", mcp.ErrorCodeInvalidRequest, "bad", nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "bad") {
		t.Fatalf("expected body to include error message, got %q", rr.Body.String())
	}
}

func TestSSEResponseSenderSendError(t *testing.T) {
	rr := httptest.NewRecorder()
	session := &SSESession{
		ID:      "session-1",
		writer:  rr,
		flusher: rr,
		nextEventID: func() string {
			return "session-1:1"
		},
	}
	sender := &SSEResponseSender{session: session}

	err := sender.SendError("id-1", mcp.ErrorCodeInternalError, "boom", nil)
	if err != nil {
		t.Fatalf("SendError failed: %v", err)
	}
	if !strings.Contains(rr.Body.String(), "boom") {
		t.Fatalf("expected SSE error payload in output, got %q", rr.Body.String())
	}
}
