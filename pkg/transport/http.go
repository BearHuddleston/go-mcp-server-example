// Package transport provides MCP transport implementations.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// HTTPTransport implements Transport for HTTP with SSE support
type HTTPTransport struct {
	port     int
	server   *http.Server
	sessions map[string]*SSESession
	mu       sync.RWMutex
	config   *config.Config
}

// HTTPResponseSender implements ResponseSender for HTTP responses
type HTTPResponseSender struct {
	writer http.ResponseWriter
	sent   bool
	mu     sync.Mutex
}

func (h *HTTPResponseSender) SendResponse(response mcp.Response) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.sent {
		return fmt.Errorf("response already sent")
	}
	
	h.writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(h.writer).Encode(response)
	h.sent = true
	return err
}

func (h *HTTPResponseSender) SendError(id any, code int, message string, data any) error {
	errorResp := &mcp.ErrorResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	response := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Error:   errorResp,
	}
	return h.SendResponse(response)
}

// SSEResponseSender implements ResponseSender for SSE streams
type SSEResponseSender struct {
	session *SSESession
}

func (s *SSEResponseSender) SendResponse(response mcp.Response) error {
	// Send as normal JSON-RPC message without event type
	return s.session.sendEvent("", response)
}

func (s *SSEResponseSender) SendError(id any, code int, message string, data any) error {
	return s.session.sendError(id, code, message, data)
}

type SSESession struct {
	ID       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	eventID  int
	mu       sync.Mutex
	closed   bool
}

// NewHTTP creates a new HTTP transport
func NewHTTP(cfg *config.Config) *HTTPTransport {
	return &HTTPTransport{
		port:     cfg.HTTPPort,
		sessions: make(map[string]*SSESession),
		config:   cfg,
	}
}

func (t *HTTPTransport) Start(ctx context.Context, server mcp.Server) error {
	mux := http.NewServeMux()
	
	// Add CORS and security middleware
	handler := t.corsMiddleware(t.securityMiddleware(mux))
	
	// MCP endpoint for POST and GET requests
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			t.handlePost(ctx, server, w, r)
		case http.MethodGet:
			t.handleGet(ctx, server, w, r)
		case http.MethodOptions:
			// CORS preflight handled by middleware
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	
	t.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", t.port),
		Handler:      handler,
		ReadTimeout:  t.config.ReadTimeout,
		WriteTimeout: t.config.WriteTimeout,
		IdleTimeout:  t.config.IdleTimeout,
	}
	
	log.Printf("Starting HTTP transport on port %d...", t.port)
	log.Printf("MCP endpoint: http://localhost:%d/mcp", t.port)
	
	// Start server in goroutine
	go func() {
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	
	// Wait for context cancellation
	<-ctx.Done()
	log.Println("HTTP transport shutting down")
	return t.Stop()
}

func (t *HTTPTransport) Stop() error {
	// Close all SSE sessions
	t.mu.Lock()
	for _, session := range t.sessions {
		session.close()
	}
	t.sessions = make(map[string]*SSESession)
	t.mu.Unlock()
	
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.config.ShutdownTimeout)
		defer cancel()
		return t.server.Shutdown(ctx)
	}
	return nil
}

func (t *HTTPTransport) handlePost(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request) {
	// Ensure UTF-8 encoding for request body
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	
	// Read request body
	var req mcp.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.sendError(w, -1, mcp.ErrorCodeParseError, "Parse error", err.Error())
		return
	}
	
	// Check Accept header to determine response type  
	acceptHeader := r.Header.Get("Accept")
	wantsSSE := strings.Contains(acceptHeader, "text/event-stream")
	wantsJSON := strings.Contains(acceptHeader, "application/json")
	
	// MCP specification requires clients to accept both types
	if !wantsJSON && !wantsSSE {
		t.sendError(w, req.ID, mcp.ErrorCodeInvalidRequest, "Accept header must include application/json and/or text/event-stream", nil)
		return
	}
	
	// Validate request
	if req.JSONRPC != mcp.JSONRPCVersion {
		t.sendError(w, req.ID, mcp.ErrorCodeInvalidRequest, "Invalid JSON-RPC version", nil)
		return
	}
	
	// Handle notifications (no response expected)
	if req.ID == nil {
		log.Printf("Received notification: %s", req.Method)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	
	// If client wants SSE and this is a request, start SSE stream
	if wantsSSE && req.ID != nil {
		t.handleSSERequest(ctx, server, w, r, req)
		return
	}
	
	// Handle regular JSON response
	t.handleJSONRequest(ctx, server, w, req)
}

func (t *HTTPTransport) handleGet(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request) {
	_ = server // Server not used for GET but kept for consistency
	// GET is used to open SSE streams or resume connections
	session := t.startSSEStream(w, r)
	if session == nil {
		return
	}
	
	// Keep the connection alive until context is cancelled
	<-ctx.Done()
	
	// Clean up session
	t.mu.Lock()
	delete(t.sessions, session.ID)
	t.mu.Unlock()
}

func (t *HTTPTransport) handleJSONRequest(ctx context.Context, server mcp.Server, w http.ResponseWriter, req mcp.Request) {
	// Create request context with timeout and HTTP response sender
	reqCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout)
	defer cancel()
	
	// Inject HTTP response sender into context
	httpSender := &HTTPResponseSender{writer: w}
	reqCtx = context.WithValue(reqCtx, mcp.ResponseSenderKey, httpSender)
	
	// Process request
	if err := server.HandleRequest(reqCtx, req); err != nil {
		log.Printf("Error handling request: %v", err)
		if !httpSender.sent {
			t.sendError(w, req.ID, mcp.ErrorCodeInternalError, "Internal error", err.Error())
		}
		return
	}
	
	// If no response was sent (shouldn't happen with proper request handling),
	// send a default error
	if !httpSender.sent {
		t.sendError(w, req.ID, mcp.ErrorCodeInternalError, "No response generated", nil)
	}
}

func (t *HTTPTransport) handleSSERequest(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request, req mcp.Request) {
	session := t.startSSEStream(w, r)
	if session == nil {
		return
	}
	
	// Process the request with SSE response sender
	reqCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout)
	defer cancel()
	
	// Inject SSE response sender and session ID into context
	sseSender := &SSEResponseSender{session: session}
	reqCtx = context.WithValue(reqCtx, mcp.ResponseSenderKey, sseSender)
	reqCtx = context.WithValue(reqCtx, mcp.SessionIDKey, session.ID)
	
	if err := server.HandleRequest(reqCtx, req); err != nil {
		log.Printf("Error handling SSE request: %v", err)
		session.sendError(req.ID, mcp.ErrorCodeInternalError, "Internal error", err.Error())
	}
}

func (t *HTTPTransport) startSSEStream(w http.ResponseWriter, r *http.Request) *SSESession {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return nil
	}
	
	// Set SSE headers with UTF-8 encoding
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Check for Last-Event-ID for connection resumption
	lastEventID := r.Header.Get("Last-Event-ID")
	eventID := 0
	if lastEventID != "" {
		if id, err := strconv.Atoi(lastEventID); err == nil {
			eventID = id + 1
		}
	}
	
	// Check for existing session ID from Mcp-Session-Id header
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		// Create new session
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	session := &SSESession{
		ID:      sessionID,
		writer:  w,
		flusher: flusher,
		eventID: eventID,
	}
	
	// Store session
	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()
	
	// Set session ID header for client
	w.Header().Set("Mcp-Session-Id", sessionID)
	
	// Send initial connection event
	session.sendEvent("connected", map[string]string{
		"sessionId": sessionID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	
	return session
}

func (t *HTTPTransport) sendError(w http.ResponseWriter, id any, code int, message string, data any) {
	errorResp := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Error: &mcp.ErrorResponse{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(errorResp)
}

func (s *SSESession) sendEvent(eventType string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return fmt.Errorf("session closed")
	}
	
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	// Write SSE event - ensure UTF-8 encoding
	fmt.Fprintf(s.writer, "id: %d\n", s.eventID)
	if eventType != "" {
		fmt.Fprintf(s.writer, "event: %s\n", eventType)
	}
	
	// Handle multi-line data properly for SSE format
	dataStr := string(dataBytes)
	for line := range strings.SplitSeq(dataStr, "\n") {
		fmt.Fprintf(s.writer, "data: %s\n", line)
	}
	fmt.Fprintf(s.writer, "\n")
	
	s.flusher.Flush()
	s.eventID++
	
	return nil
}

func (s *SSESession) sendError(id any, code int, message string, data any) error {
	errorResp := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Error: &mcp.ErrorResponse{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	
	// Send error as a normal JSON-RPC response, not as an "error" event type
	return s.sendEvent("", errorResp)
}

func (s *SSESession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

func (t *HTTPTransport) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Last-Event-ID, Mcp-Session-Id")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		next.ServeHTTP(w, r)
	})
}

func (t *HTTPTransport) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Validate Origin for security (DNS rebinding protection)
		origin := r.Header.Get("Origin")
		if origin != "" {
			// Check if this is a local development request
			isLocal := strings.Contains(r.Host, "localhost") || 
					  strings.Contains(r.Host, "127.0.0.1") ||
					  strings.Contains(r.Host, "::1")
			
			if !isLocal {
				// In production, validate against allowed origins to prevent DNS rebinding
				// For now, log and allow but this should be configurable
				log.Printf("Warning: Request from external origin: %s to host: %s", origin, r.Host)
				// TODO: Implement allowlist checking: if !isOriginAllowed(origin) { http.Error(...) }
			}
		}
		
		next.ServeHTTP(w, r)
	})
}