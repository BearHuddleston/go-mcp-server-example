// Package transport provides MCP transport implementations.
package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// HTTPTransport implements Transport for HTTP with SSE support
type HTTPTransport struct {
	port          int
	server        *http.Server
	sessions      map[string]*SSESession
	knownSessions map[string]struct{}
	mu            sync.RWMutex
	config        *config.Config
	originRegexes []*regexp.Regexp
}

// HTTPResponseSender implements ResponseSender for HTTP responses
type HTTPResponseSender struct {
	writer    http.ResponseWriter
	sent      bool
	mu        sync.Mutex
	sessionID string
}

func (h *HTTPResponseSender) SendResponse(response mcp.Response) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.sent {
		return errors.New("response already sent")
	}

	if h.sessionID != "" {
		h.writer.Header().Set(mcp.SessionIDHeader, h.sessionID)
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
	ID      string
	writer  http.ResponseWriter
	flusher http.Flusher
	eventID int
	mu      sync.Mutex
	closed  bool
}

// NewHTTP creates a new HTTP transport
func NewHTTP(cfg *config.Config) *HTTPTransport {
	t := &HTTPTransport{
		port:          cfg.HTTPPort,
		sessions:      make(map[string]*SSESession),
		knownSessions: make(map[string]struct{}),
		config:        cfg,
	}

	// Pre-compile regex patterns for origin validation
	for _, allowed := range cfg.AllowedOrigins {
		if allowed == "*" {
			// Wildcard - skip regex, handled separately
			continue
		}

		// Support port wildcards (e.g., "http://localhost:*")
		pattern := "^" + strings.ReplaceAll(allowed, "*", ".*") + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			slog.Warn("invalid origin pattern; skipping", "pattern", allowed, "error", err)
			continue
		}
		t.originRegexes = append(t.originRegexes, re)
	}

	return t
}

func (t *HTTPTransport) Start(ctx context.Context, server mcp.Server) error {
	mux := http.NewServeMux()

	// Add CORS and security middleware
	handler := t.corsMiddleware(t.securityMiddleware(mux))

	mux.HandleFunc("POST /mcp", func(w http.ResponseWriter, r *http.Request) {
		t.handlePost(ctx, server, w, r)
	})
	mux.HandleFunc("GET /mcp", func(w http.ResponseWriter, r *http.Request) {
		t.handleGet(ctx, server, w, r)
	})
	mux.HandleFunc("DELETE /mcp", func(w http.ResponseWriter, r *http.Request) {
		t.handleDelete(w, r)
	})
	mux.HandleFunc("OPTIONS /mcp", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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

	slog.Info("starting HTTP transport", "port", t.port)
	slog.Info("MCP endpoint", "url", fmt.Sprintf("http://localhost:%d/mcp", t.port))

	// Start server in goroutine
	go func() {
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("HTTP transport shutting down")
	return t.Stop()
}

func (t *HTTPTransport) Stop() error {
	// Close all SSE sessions
	t.mu.Lock()
	for _, session := range t.sessions {
		session.close()
	}
	t.sessions = make(map[string]*SSESession)
	t.knownSessions = make(map[string]struct{})
	t.mu.Unlock()

	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), t.config.ShutdownTimeout)
		defer cancel()
		return t.server.Shutdown(ctx)
	}
	return nil
}

func (t *HTTPTransport) handlePost(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		t.sendErrorWithStatus(w, -1, mcp.ErrorCodeParseError, "Parse error", err.Error(), http.StatusBadRequest)
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		t.sendErrorWithStatus(w, -1, mcp.ErrorCodeInvalidRequest, "Request body cannot be empty", nil, http.StatusBadRequest)
		return
	}
	if strings.HasPrefix(trimmed, "[") {
		t.sendErrorWithStatus(w, -1, mcp.ErrorCodeInvalidRequest, "Batch requests are not supported", nil, http.StatusBadRequest)
		return
	}

	var req mcp.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		t.sendErrorWithStatus(w, -1, mcp.ErrorCodeParseError, "Parse error", err.Error(), http.StatusBadRequest)
		return
	}

	acceptHeader := r.Header.Get("Accept")
	wantsSSE := strings.Contains(acceptHeader, "text/event-stream")
	wantsJSON := strings.Contains(acceptHeader, "application/json")
	if !wantsJSON && !wantsSSE {
		t.sendErrorWithStatus(w, req.ID, mcp.ErrorCodeInvalidRequest, "Accept header must include application/json and/or text/event-stream", nil, http.StatusBadRequest)
		return
	}

	if req.JSONRPC != mcp.JSONRPCVersion {
		t.sendErrorWithStatus(w, req.ID, mcp.ErrorCodeInvalidRequest, "Invalid JSON-RPC version", nil, http.StatusBadRequest)
		return
	}

	if req.Method == "" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	sessionID, err := t.resolveSessionForRequest(r, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errUnknownSession) {
			status = http.StatusNotFound
		}
		t.sendErrorWithStatus(w, req.ID, mcp.ErrorCodeInvalidRequest, err.Error(), nil, status)
		return
	}

	if req.ID == nil {
		slog.Info("received notification", "method", req.Method)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if wantsSSE {
		t.handleSSERequest(ctx, server, w, r, req, sessionID)
		return
	}

	t.handleJSONRequest(ctx, server, w, req, sessionID)
}

func (t *HTTPTransport) handleGet(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request) {
	_ = server
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.Header.Get(mcp.SessionIDHeader)
	if sessionID == "" {
		http.Error(w, "Missing MCP session header", http.StatusBadRequest)
		return
	}

	if err := t.ensureProtocolVersion(r.Header.Get(mcp.ProtocolVersionHeader)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !t.sessionExists(sessionID) {
		http.Error(w, "Unknown session", http.StatusNotFound)
		return
	}

	session := t.startSSEStream(w, r, sessionID)
	if session == nil {
		return
	}

	<-ctx.Done()

	t.mu.Lock()
	delete(t.sessions, session.ID)
	t.mu.Unlock()
}

func (t *HTTPTransport) handleDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get(mcp.SessionIDHeader)
	if sessionID == "" {
		http.Error(w, "Missing MCP session header", http.StatusBadRequest)
		return
	}

	if err := t.ensureProtocolVersion(r.Header.Get(mcp.ProtocolVersionHeader)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t.mu.Lock()
	_, known := t.knownSessions[sessionID]
	delete(t.knownSessions, sessionID)
	if session, ok := t.sessions[sessionID]; ok {
		session.close()
		delete(t.sessions, sessionID)
	}
	t.mu.Unlock()

	if !known {
		http.Error(w, "Unknown session", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var errUnknownSession = errors.New("unknown session")

func (t *HTTPTransport) resolveSessionForRequest(r *http.Request, req mcp.Request) (string, error) {
	isInitialize := req.Method == "initialize"

	if !isInitialize {
		if err := t.ensureProtocolVersion(r.Header.Get(mcp.ProtocolVersionHeader)); err != nil {
			return "", err
		}
	}

	sessionID := r.Header.Get(mcp.SessionIDHeader)
	if isInitialize {
		if sessionID != "" {
			if !t.sessionExists(sessionID) {
				return "", errUnknownSession
			}
			return sessionID, nil
		}
		created, err := generateSessionID()
		if err != nil {
			return "", fmt.Errorf("failed to generate session id: %w", err)
		}
		t.registerSession(created)
		return created, nil
	}

	if sessionID == "" {
		return "", errors.New("missing MCP session header")
	}
	if !t.sessionExists(sessionID) {
		return "", errUnknownSession
	}
	return sessionID, nil
}

func (t *HTTPTransport) ensureProtocolVersion(version string) error {
	if version == "" || version == mcp.LegacyProtocolVersion || version == mcp.ProtocolVersion {
		return nil
	}
	return fmt.Errorf("unsupported MCP protocol version: %s", version)
}

func (t *HTTPTransport) registerSession(sessionID string) {
	t.mu.Lock()
	t.knownSessions[sessionID] = struct{}{}
	t.mu.Unlock()
}

func (t *HTTPTransport) sessionExists(sessionID string) bool {
	t.mu.RLock()
	_, ok := t.knownSessions[sessionID]
	t.mu.RUnlock()
	return ok
}

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (t *HTTPTransport) handleJSONRequest(ctx context.Context, server mcp.Server, w http.ResponseWriter, req mcp.Request, sessionID string) {
	reqCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout)
	defer cancel()

	httpSender := &HTTPResponseSender{writer: w, sessionID: sessionID}
	reqCtx = context.WithValue(reqCtx, mcp.ResponseSenderKey, httpSender)
	if sessionID != "" {
		reqCtx = context.WithValue(reqCtx, mcp.SessionIDKey, sessionID)
	}

	if err := server.HandleRequest(reqCtx, req); err != nil {
		slog.Error("error handling request", "error", err)
		if !httpSender.sent {
			t.sendErrorWithStatus(w, req.ID, mcp.ErrorCodeInternalError, "Internal error", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if !httpSender.sent {
		t.sendErrorWithStatus(w, req.ID, mcp.ErrorCodeInternalError, "No response generated", nil, http.StatusInternalServerError)
	}
}

func (t *HTTPTransport) handleSSERequest(ctx context.Context, server mcp.Server, w http.ResponseWriter, r *http.Request, req mcp.Request, sessionID string) {
	session := t.startSSEStream(w, r, sessionID)
	if session == nil {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout)
	defer cancel()

	sseSender := &SSEResponseSender{session: session}
	reqCtx = context.WithValue(reqCtx, mcp.ResponseSenderKey, sseSender)
	reqCtx = context.WithValue(reqCtx, mcp.SessionIDKey, session.ID)

	if err := server.HandleRequest(reqCtx, req); err != nil {
		slog.Error("error handling SSE request", "error", err)
		session.sendError(req.ID, mcp.ErrorCodeInternalError, "Internal error", err.Error())
	}
}

func (t *HTTPTransport) startSSEStream(w http.ResponseWriter, r *http.Request, sessionID string) *SSESession {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set(mcp.SessionIDHeader, sessionID)

	lastEventID := r.Header.Get("Last-Event-ID")
	eventID := 0
	if lastEventID != "" {
		if id, err := strconv.Atoi(lastEventID); err == nil {
			eventID = id + 1
		}
	}

	session := &SSESession{
		ID:      sessionID,
		writer:  w,
		flusher: flusher,
		eventID: eventID,
	}

	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	session.sendEvent("connected", map[string]string{
		"sessionId": sessionID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	return session
}

func (t *HTTPTransport) sendError(w http.ResponseWriter, id any, code int, message string, data any) {
	t.sendErrorWithStatus(w, id, code, message, data, http.StatusBadRequest)
}

func (t *HTTPTransport) sendErrorWithStatus(w http.ResponseWriter, id any, code int, message string, data any, status int) {
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
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResp)
}

func (s *SSESession) sendEvent(eventType string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("session closed")
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
		origin := r.Header.Get("Origin")

		// Set CORS headers based on allowed origins
		if origin != "" && t.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// For same-origin requests or non-browser clients, don't set Access-Control-Allow-Origin
			// This avoids browser warnings about wildcard with credentials
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Last-Event-ID, MCP-Session-Id, MCP-Protocol-Version")
		w.Header().Set("Access-Control-Max-Age", "86400")

		next.ServeHTTP(w, r)
	})
}

func (t *HTTPTransport) isOriginAllowed(origin string) bool {
	// Check against configured allowed origins
	for _, allowed := range t.config.AllowedOrigins {
		// Support wildcard patterns
		if allowed == "*" {
			return true
		}
	}

	// Check against pre-compiled regex patterns
	for _, re := range t.originRegexes {
		if re.MatchString(origin) {
			return true
		}
	}

	// Fallback: check if it's a localhost request for development
	if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") || strings.Contains(origin, "::1") {
		return true
	}

	return false
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
			if !t.isOriginAllowed(origin) {
				slog.Warn("rejected request from disallowed origin", "origin", origin)
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
