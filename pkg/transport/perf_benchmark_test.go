package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BearHuddleston/mcp-server-template/pkg/config"
	"github.com/BearHuddleston/mcp-server-template/pkg/mcp"
)

func BenchmarkClassifyJSONRPCMessageRequest(b *testing.B) {
	b.ReportAllocs()
	raw := json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	for b.Loop() {
		kind, req, _, err := classifyJSONRPCMessage(raw)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if kind != messageKindRequest || req.Method != "tools/list" {
			b.Fatalf("unexpected decode result: kind=%s method=%s", kind, req.Method)
		}
	}
}

func BenchmarkClassifyJSONRPCMessageResponse(b *testing.B) {
	b.ReportAllocs()
	raw := json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)

	for b.Loop() {
		kind, _, _, err := classifyJSONRPCMessage(raw)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		if kind != messageKindResponse {
			b.Fatalf("unexpected kind: %s", kind)
		}
	}
}

func BenchmarkParseAcceptTypes(b *testing.B) {
	b.ReportAllocs()
	header := "application/json;q=1.0, text/event-stream;q=0.9"

	for b.Loop() {
		wantsJSON, wantsSSE := parseAcceptTypes(header)
		if !wantsJSON || !wantsSSE {
			b.Fatalf("unexpected accept parse result: json=%v sse=%v", wantsJSON, wantsSSE)
		}
	}
}

func BenchmarkIsOriginAllowed(b *testing.B) {
	b.ReportAllocs()
	tx := newHTTPTransportForTest(func(cfg *config.Config) {
		cfg.AllowedOrigins = []string{"https://api.example.com", "http://localhost:*"}
	})
	origin := "https://api.example.com"

	for b.Loop() {
		if !tx.isOriginAllowed(origin) {
			b.Fatal("expected origin to be allowed")
		}
	}
}

func BenchmarkHandlePostToolsList(b *testing.B) {
	b.ReportAllocs()
	tx := newHTTPTransportForTest()
	tx.registerSession("session-1")
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set(mcp.SessionIDHeader, "session-1")
		req.Header.Set(mcp.ProtocolVersionHeader, mcp.ProtocolVersion)

		rr := httptest.NewRecorder()
		tx.handlePost(context.Background(), &httpMockServer{}, rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rr.Code)
		}
	}
}

func BenchmarkStdioHandleMessageRequest(b *testing.B) {
	b.ReportAllocs()
	tx := NewStdio(config.New())
	out := &bytes.Buffer{}
	tx.output = out
	req := `{"jsonrpc":"2.0","id":1,"method":"test"}`

	b.ResetTimer()
	for b.Loop() {
		out.Reset()
		if err := tx.handleMessage(context.Background(), &mockServer{}, req); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
