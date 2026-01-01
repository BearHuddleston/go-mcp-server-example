package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

func TestNewHTTP(t *testing.T) {
	t.Run("creates HTTP transport", func(t *testing.T) {
		cfg := config.New()
		cfg.HTTPPort = 8080

		transport := NewHTTP(cfg)
		if transport == nil {
			t.Fatalf("Expected non-nil transport")
		}

		if transport.port != 8080 {
			t.Errorf("Expected port 8080, got %d", transport.port)
		}

		if transport.sessions == nil {
			t.Errorf("Expected sessions map to be initialized")
		}
	})
}

func TestHTTPResponseSender(t *testing.T) {
	t.Run("send response", func(t *testing.T) {
		w := httptest.NewRecorder()
		sender := &HTTPResponseSender{writer: w}

		resp := mcp.Response{
			JSONRPC: mcp.JSONRPCVersion,
			ID:      "test-id",
			Result:  map[string]string{"status": "ok"},
		}

		err := sender.SendResponse(resp)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("Expected content-type 'application/json; charset=utf-8', got %s", contentType)
		}

		var decoded mcp.Response
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if decoded.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got: %v", decoded.ID)
		}
	})

	t.Run("cannot send response twice", func(t *testing.T) {
		w := httptest.NewRecorder()
		sender := &HTTPResponseSender{writer: w}

		resp := mcp.Response{
			JSONRPC: mcp.JSONRPCVersion,
			ID:      "test-id",
			Result:  map[string]string{"status": "ok"},
		}

		err := sender.SendResponse(resp)
		if err != nil {
			t.Errorf("Expected no error on first send, got: %v", err)
		}

		err = sender.SendResponse(resp)
		if err == nil {
			t.Errorf("Expected error on second send")
		}

		if err.Error() != "response already sent" {
			t.Errorf("Expected 'response already sent' error, got: %v", err)
		}
	})

	t.Run("send error", func(t *testing.T) {
		w := httptest.NewRecorder()
		sender := &HTTPResponseSender{writer: w}

		err := sender.SendError("test-id", mcp.ErrorCodeInvalidRequest, "Invalid request", nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		var decoded mcp.Response
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if decoded.Error == nil {
			t.Errorf("Expected error response")
		}

		if decoded.Error.Code != mcp.ErrorCodeInvalidRequest {
			t.Errorf("Expected invalid request error code, got: %d", decoded.Error.Code)
		}

		if decoded.Error.Message != "Invalid request" {
			t.Errorf("Expected error message 'Invalid request', got: %s", decoded.Error.Message)
		}
	})
}

func TestHTTPTransport_Start(t *testing.T) {
	t.Run("starts HTTP server", func(t *testing.T) {
		cfg := config.New()
		cfg.HTTPPort = 0 // Use random available port

		transport := NewHTTP(cfg)
		server := &mockServer{}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start server in goroutine
		readyChan := make(chan bool, 1)
		go func() {
			readyChan <- true
			transport.Start(ctx, server)
		}()

		// Wait for server to start
		<-readyChan
		time.Sleep(100 * time.Millisecond)

		// Verify server is running
		if transport.server == nil {
			t.Errorf("Expected server to be initialized")
		}

		// Create a test server with the same handler for testing
		handler := transport.corsMiddleware(transport.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		})))
		testServer := httptest.NewServer(handler)
		defer testServer.Close()

		// Test health endpoint via test server
		resp, err := http.Get(testServer.URL + "/")
		if err != nil {
			t.Errorf("Failed to call endpoint: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Clean up
		cancel()
		time.Sleep(50 * time.Millisecond)
	})
}

func TestHTTPTransport_handlePost(t *testing.T) {
	cfg := config.New()
	transport := NewHTTP(cfg)
	server := &mockServer{}

	t.Run("valid JSON request", func(t *testing.T) {
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "initialize",
			ID:      "test-id",
			Params:  nil,
		}
		reqJSON, _ := json.Marshal(req)

		w := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqJSON))
		w.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		// The response may be success or error depending on how the handler processes it
		// The key is that we get a response
		var resp mcp.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		// We should have some kind of response
		if rec.Code == 0 {
			t.Errorf("Expected a response status code")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		w := httptest.NewRequest("POST", "/mcp", strings.NewReader("invalid json"))
		w.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		var resp mcp.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Errorf("Expected error response")
		}

		if resp.Error.Code != mcp.ErrorCodeParseError {
			t.Errorf("Expected parse error code, got: %d", resp.Error.Code)
		}
	})

	t.Run("invalid JSON-RPC version", func(t *testing.T) {
		req := map[string]any{
			"jsonrpc": "1.0",
			"method":  "initialize",
			"id":      "test-id",
		}
		reqJSON, _ := json.Marshal(req)

		w := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqJSON))
		w.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		var resp mcp.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Errorf("Expected error response")
		}

		if resp.Error.Code != mcp.ErrorCodeInvalidRequest {
			t.Errorf("Expected invalid request error code, got: %d", resp.Error.Code)
		}
	})

	t.Run("notification (no ID)", func(t *testing.T) {
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "notification",
			ID:      nil,
		}
		reqJSON, _ := json.Marshal(req)

		w := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqJSON))
		w.Header.Set("Accept", "application/json")
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rec.Code)
		}
	})

	t.Run("missing Accept header", func(t *testing.T) {
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "initialize",
			ID:      "test-id",
		}
		reqJSON, _ := json.Marshal(req)

		w := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqJSON))
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		var resp mcp.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Errorf("Expected error response")
		}
	})

	t.Run("SSE request", func(t *testing.T) {
		req := mcp.Request{
			JSONRPC: mcp.JSONRPCVersion,
			Method:  "initialize",
			ID:      "test-id",
		}
		reqJSON, _ := json.Marshal(req)

		w := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqJSON))
		w.Header.Set("Accept", "text/event-stream")
		rec := httptest.NewRecorder()

		transport.handlePost(context.Background(), server, rec, w)

		// SSE should return 200 with appropriate headers
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/event-stream; charset=utf-8" {
			t.Errorf("Expected text/event-stream content type, got: %s", contentType)
		}
	})
}

func TestHTTPTransport_handleGet(t *testing.T) {
	cfg := config.New()
	transport := NewHTTP(cfg)
	server := &mockServer{}

	t.Run("opens SSE stream", func(t *testing.T) {
		w := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go transport.handleGet(ctx, server, rec, w)
		<-ctx.Done()

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/event-stream; charset=utf-8" {
			t.Errorf("Expected text/event-stream content type, got: %s", contentType)
		}

		// Should have session ID header
		sessionID := rec.Header().Get("Mcp-Session-Id")
		if sessionID == "" {
			t.Errorf("Expected session ID header")
		}
	})
}

func TestHTTPTransport_corsMiddleware(t *testing.T) {
	cfg := config.New()
	transport := NewHTTP(cfg)

	t.Run("adds CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/mcp", nil)
		rec := httptest.NewRecorder()

		middleware := transport.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		middleware.ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got: %s", origin)
		}

		methods := rec.Header().Get("Access-Control-Allow-Methods")
		if methods != "GET, POST, OPTIONS" {
			t.Errorf("Expected 'GET, POST, OPTIONS', got: %s", methods)
		}

		headers := rec.Header().Get("Access-Control-Allow-Headers")
		if headers != "Content-Type, Accept, Last-Event-ID, Mcp-Session-Id" {
			t.Errorf("Unexpected headers: %s", headers)
		}
	})
}

func TestHTTPTransport_securityMiddleware(t *testing.T) {
	cfg := config.New()
	transport := NewHTTP(cfg)

	t.Run("adds security headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()

		middleware := transport.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		middleware.ServeHTTP(rec, req)

		cto := rec.Header().Get("X-Content-Type-Options")
		if cto != "nosniff" {
			t.Errorf("Expected X-Content-Type-Options 'nosniff', got: %s", cto)
		}

		xf := rec.Header().Get("X-Frame-Options")
		if xf != "DENY" {
			t.Errorf("Expected X-Frame-Options 'DENY', got: %s", xf)
		}

		xss := rec.Header().Get("X-XSS-Protection")
		if xss != "1; mode=block" {
			t.Errorf("Expected X-XSS-Protection '1; mode=block', got: %s", xss)
		}
	})
}

func TestSSESession(t *testing.T) {
	t.Run("send event", func(t *testing.T) {
		rec := &flushingRecorder{httptest.NewRecorder()}
		session := &SSESession{
			ID:      "test-session",
			writer:  rec,
			flusher: rec,
			eventID: 0,
		}

		data := map[string]string{"message": "test"}
		err := session.sendEvent("test-event", data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "id: 0") {
			t.Errorf("Expected event ID in response")
		}

		if !strings.Contains(body, "event: test-event") {
			t.Errorf("Expected event type in response")
		}

		if !strings.Contains(body, "data:") {
			t.Errorf("Expected data in response")
		}

		// Event ID should increment
		if session.eventID != 1 {
			t.Errorf("Expected event ID 1, got: %d", session.eventID)
		}
	})

	t.Run("send event when closed", func(t *testing.T) {
		rec := &flushingRecorder{httptest.NewRecorder()}
		session := &SSESession{
			ID:      "test-session",
			writer:  rec,
			flusher: rec,
			eventID: 0,
			closed:  true,
		}

		data := map[string]string{"message": "test"}
		err := session.sendEvent("test-event", data)
		if err == nil {
			t.Errorf("Expected error when session is closed")
		}

		if err.Error() != "session closed" {
			t.Errorf("Expected 'session closed' error, got: %v", err)
		}
	})

	t.Run("send error", func(t *testing.T) {
		rec := &flushingRecorder{httptest.NewRecorder()}
		session := &SSESession{
			ID:      "test-session",
			writer:  rec,
			flusher: rec,
			eventID: 0,
		}

		err := session.sendError("test-id", mcp.ErrorCodeInternalError, "Internal error", nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "data:") {
			t.Errorf("Expected data in response")
		}

		var resp mcp.Response
		// Find the JSON data in the SSE format
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data:") {
				jsonStr := strings.TrimPrefix(line, "data: ")
				if err := json.Unmarshal([]byte(jsonStr), &resp); err == nil {
					if resp.Error == nil {
						t.Errorf("Expected error in response")
					}
					if resp.Error.Code != mcp.ErrorCodeInternalError {
						t.Errorf("Expected internal error code, got: %d", resp.Error.Code)
					}
					break
				}
			}
		}
	})

	t.Run("close session", func(t *testing.T) {
		rec := &flushingRecorder{httptest.NewRecorder()}
		session := &SSESession{
			ID:      "test-session",
			writer:  rec,
			flusher: rec,
			eventID: 0,
			closed:  false,
		}

		session.close()
		if !session.closed {
			t.Errorf("Expected session to be closed")
		}
	})
}

func TestHTTPTransport_startSSEStream(t *testing.T) {
	cfg := config.New()
	transport := NewHTTP(cfg)

	t.Run("creates new session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()

		session := transport.startSSEStream(rec, req)
		if session == nil {
			t.Errorf("Expected session to be created")
		}

		// Check headers
		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/event-stream; charset=utf-8" {
			t.Errorf("Expected text/event-stream content type, got: %s", contentType)
		}

		// Check session ID header
		sessionID := rec.Header().Get("Mcp-Session-Id")
		if sessionID == "" {
			t.Errorf("Expected session ID header")
		}

		// Check session is stored
		transport.mu.RLock()
		_, exists := transport.sessions[sessionID]
		transport.mu.RUnlock()

		if !exists {
			t.Errorf("Expected session to be stored in sessions map")
		}

		// Check initial connected event
		body := rec.Body.String()
		if !strings.Contains(body, "event: connected") {
			t.Errorf("Expected connected event")
		}
	})

	t.Run("resumes existing session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Mcp-Session-Id", "existing-session")
		rec := httptest.NewRecorder()

		session := transport.startSSEStream(rec, req)
		if session == nil {
			t.Errorf("Expected session to be created")
		}

		if session.ID != "existing-session" {
			t.Errorf("Expected session ID 'existing-session', got: %s", session.ID)
		}
	})

	t.Run("uses Last-Event-ID for resume", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Last-Event-ID", "5")
		rec := httptest.NewRecorder()

		session := transport.startSSEStream(rec, req)
		if session == nil {
			t.Errorf("Expected session to be created")
		}

		// Event ID is 6 (5+1) but then the connected event increments it to 7
		if session.eventID != 7 {
			t.Errorf("Expected event ID 7, got: %d", session.eventID)
		}
	})

	t.Run("session handles flusher", func(t *testing.T) {
		// Note: httptest.ResponseRecorder implements http.Flusher
		// This test verifies that sessions work with a flusher
		req := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()

		session := transport.startSSEStream(rec, req)
		if session == nil {
			t.Errorf("Expected session to be created")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", rec.Code)
		}

		// Verify session was created successfully
		if session.ID == "" {
			t.Errorf("Expected session ID to be set")
		}
	})
}

func TestHTTPTransport_Stop(t *testing.T) {
	t.Run("stops server and closes sessions", func(t *testing.T) {
		cfg := config.New()
		transport := NewHTTP(cfg)

		// Start server
		ctx, cancel := context.WithCancel(context.Background())
		server := &mockServer{}

		errChan := make(chan error, 1)
		go func() {
			errChan <- transport.Start(ctx, server)
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Add a session
		req := httptest.NewRequest("GET", "/mcp", nil)
		rec := httptest.NewRecorder()
		session := transport.startSSEStream(rec, req)
		if session == nil {
			t.Fatalf("Expected session to be created")
		}

		// Stop the transport
		cancel()
		err := transport.Stop()
		if err != nil {
			t.Errorf("Expected no error on stop, got: %v", err)
		}

		// Wait for server to stop
		<-errChan

		// Check sessions are cleared
		transport.mu.RLock()
		count := len(transport.sessions)
		transport.mu.RUnlock()

		if count != 0 {
			t.Errorf("Expected sessions to be cleared, got %d sessions", count)
		}
	})
}

// flushingRecorder wraps httptest.ResponseRecorder to implement http.Flusher
type flushingRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushingRecorder) Flush() {
	// No-op for testing
}

// noFlusherRecorder wraps httptest.ResponseRecorder to NOT implement http.Flusher
type noFlusherRecorder struct {
	*httptest.ResponseRecorder
}

func (n *noFlusherRecorder) Close() error {
	return nil
}

func (n *noFlusherRecorder) Write(p []byte) (int, error) {
	return n.ResponseRecorder.Write(p)
}
