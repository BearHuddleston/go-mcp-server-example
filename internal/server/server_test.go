package server

import (
	"context"
	"errors"
	"testing"

	"github.com/BearHuddleston/mcp-server-example/pkg/config"
	"github.com/BearHuddleston/mcp-server-example/pkg/handlers"
	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
	"github.com/BearHuddleston/mcp-server-example/pkg/spec"
)

type testToolHandler struct {
	listErr error
	callErr error
	resp    mcp.ToolResponse
	last    mcp.ToolCallParams
}

func (h *testToolHandler) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if h.listErr != nil {
		return nil, h.listErr
	}
	return []mcp.Tool{{Name: "toolA"}}, nil
}

func (h *testToolHandler) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	h.last = params
	if h.callErr != nil {
		return mcp.ToolResponse{}, h.callErr
	}
	if h.resp.Content == nil {
		h.resp = mcp.ToolResponse{Content: []mcp.ContentItem{{Type: "text", Text: "ok"}}}
	}
	return h.resp, nil
}

type testResourceHandler struct {
	listErr error
	readErr error
	resp    mcp.ResourceResponse
	last    mcp.ResourceParams
}

func (h *testResourceHandler) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	if h.listErr != nil {
		return nil, h.listErr
	}
	return []mcp.Resource{{URI: "catalog://items", Name: "catalog"}}, nil
}

func (h *testResourceHandler) ReadResource(ctx context.Context, params mcp.ResourceParams) (mcp.ResourceResponse, error) {
	h.last = params
	if h.readErr != nil {
		return mcp.ResourceResponse{}, h.readErr
	}
	if h.resp.Contents == nil {
		h.resp = mcp.ResourceResponse{Contents: []mcp.ResourceContent{{URI: params.URI, Text: "{}"}}}
	}
	return h.resp, nil
}

type testPromptHandler struct {
	listErr error
	getErr  error
	resp    mcp.PromptResponse
	last    mcp.PromptParams
}

func (h *testPromptHandler) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	if h.listErr != nil {
		return nil, h.listErr
	}
	return []mcp.Prompt{{Name: "promptA"}}, nil
}

func (h *testPromptHandler) GetPrompt(ctx context.Context, params mcp.PromptParams) (mcp.PromptResponse, error) {
	h.last = params
	if h.getErr != nil {
		return mcp.PromptResponse{}, h.getErr
	}
	if h.resp.Messages == nil {
		h.resp = mcp.PromptResponse{Messages: []mcp.PromptMessage{{Role: "user", Content: mcp.MessageContent{Type: "text", Text: "ok"}}}}
	}
	return h.resp, nil
}

type captureSender struct {
	response  *mcp.Response
	errorID   any
	errorCode int
	errorMsg  string
	errorData any
}

func (s *captureSender) SendResponse(response mcp.Response) error {
	copy := response
	s.response = &copy
	return nil
}

func (s *captureSender) SendError(id any, code int, message string, data any) error {
	s.errorID = id
	s.errorCode = code
	s.errorMsg = message
	s.errorData = data
	return nil
}

func newTestConfig() *config.Config {
	return &config.Config{ServerName: "test", ServerVersion: "1.0.0"}
}

func newServerWithHandlers(t *testing.T) (*Server, *testToolHandler, *testResourceHandler, *testPromptHandler) {
	t.Helper()

	tool := &testToolHandler{}
	resource := &testResourceHandler{}
	prompt := &testPromptHandler{}

	srv, err := New(newTestConfig(), tool, resource, prompt)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	return srv, tool, resource, prompt
}

func TestNewValidation(t *testing.T) {
	tool := &testToolHandler{}
	resource := &testResourceHandler{}
	prompt := &testPromptHandler{}

	if _, err := New(nil, tool, resource, prompt); err == nil {
		t.Fatal("expected error for nil config")
	}
	if _, err := New(newTestConfig(), nil, resource, prompt); err == nil {
		t.Fatal("expected error for nil tool handler")
	}
	if _, err := New(newTestConfig(), tool, nil, prompt); err == nil {
		t.Fatal("expected error for nil resource handler")
	}
	if _, err := New(newTestConfig(), tool, resource, nil); err == nil {
		t.Fatal("expected error for nil prompt handler")
	}
}

func TestHandleRequestDispatchSuccess(t *testing.T) {
	srv, tool, resource, prompt := newServerWithHandlers(t)

	tests := []mcp.Request{
		{JSONRPC: mcp.JSONRPCVersion, Method: "initialize", ID: 1},
		{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 2},
		{JSONRPC: mcp.JSONRPCVersion, Method: "tools/call", ID: 3, Params: map[string]any{"name": "toolA", "arguments": map[string]any{"k": "v"}}},
		{JSONRPC: mcp.JSONRPCVersion, Method: "resources/list", ID: 4},
		{JSONRPC: mcp.JSONRPCVersion, Method: "resources/read", ID: 5, Params: map[string]any{"uri": "catalog://items"}},
		{JSONRPC: mcp.JSONRPCVersion, Method: "prompts/list", ID: 6},
		{JSONRPC: mcp.JSONRPCVersion, Method: "prompts/get", ID: 7, Params: map[string]any{"name": "promptA", "arguments": map[string]any{"k": "v"}}},
		{JSONRPC: mcp.JSONRPCVersion, Method: "ping", ID: 8},
	}

	for _, req := range tests {
		sender := &captureSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		if err := srv.HandleRequest(ctx, req); err != nil {
			t.Fatalf("HandleRequest failed for %s: %v", req.Method, err)
		}
		if sender.response == nil {
			t.Fatalf("expected response for %s", req.Method)
		}
	}

	if tool.last.Name != "toolA" || tool.last.Arguments["k"] != "v" {
		t.Fatalf("unexpected tool params captured: %+v", tool.last)
	}
	if resource.last.URI != "catalog://items" {
		t.Fatalf("unexpected resource params captured: %+v", resource.last)
	}
	if prompt.last.Name != "promptA" || prompt.last.Arguments["k"] != "v" {
		t.Fatalf("unexpected prompt params captured: %+v", prompt.last)
	}
}

func TestHandleRequestToolsListWithSpecBackedCatalog(t *testing.T) {
	sp := &spec.Spec{
		SchemaVersion: "v1",
		Runtime:       spec.RuntimeSpec{TransportType: "stdio"},
		Items: []spec.ItemSpec{{
			"item_name": "Template Bundle",
			"score":     9,
			"track":     "starter",
		}},
		Tools: []spec.ToolSpec{
			{Mode: "list_items", Name: "listCatalog", Description: "List item names", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}, Required: []string{}}},
			{Mode: "get_item_details", Name: "fetchDetails", Description: "Fetch item details", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{"item_name": map[string]string{"type": "string"}}, Required: []string{"item_name"}}},
		},
		Resources: []spec.ResourceSpec{{Mode: "catalog_items", URI: "catalog://custom-items", Name: "custom-catalog"}},
		Prompts: []spec.PromptSpec{
			{Mode: "plan_recommendation", Name: "buildPlan", Description: "Build recommendation", Template: "Plan for team%s%s"},
			{Mode: "item_brief", Name: "quickBrief", Description: "Create brief", Template: "Brief for %s"},
		},
	}

	catalog, err := handlers.NewCatalogFromSpec(sp)
	if err != nil {
		t.Fatalf("NewCatalogFromSpec failed: %v", err)
	}

	srv, err := New(newTestConfig(), catalog, catalog, catalog)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	sender := &captureSender{}
	ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
	req := mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 99}

	if err := srv.HandleRequest(ctx, req); err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if sender.response == nil {
		t.Fatal("expected tools/list response")
	}

	result, ok := sender.response.Result.(map[string][]mcp.Tool)
	if !ok {
		t.Fatalf("expected map[string][]mcp.Tool result, got %T", sender.response.Result)
	}
	toolsAny := result["tools"]
	if len(toolsAny) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(toolsAny))
	}
	found := map[string]bool{"listCatalog": false, "fetchDetails": false}
	for _, tool := range toolsAny {
		if _, ok := found[tool.Name]; ok {
			found[tool.Name] = true
		}
	}
	for name, ok := range found {
		if !ok {
			t.Fatalf("expected tool %q not found in %+v", name, toolsAny)
		}
	}
}

func TestHandleRequestErrors(t *testing.T) {
	tool := &testToolHandler{listErr: errors.New("list fail")}
	resource := &testResourceHandler{readErr: errors.New("read fail")}
	prompt := &testPromptHandler{getErr: errors.New("prompt fail")}
	srv, err := New(newTestConfig(), tool, resource, prompt)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	tests := []struct {
		name       string
		req        mcp.Request
		expectCode int
	}{
		{name: "unknown method", req: mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "nope", ID: 1}, expectCode: mcp.ErrorCodeMethodNotFound},
		{name: "tools list error", req: mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/list", ID: 2}, expectCode: mcp.ErrorCodeInternalError},
		{name: "tools call bad params", req: mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "tools/call", ID: 3, Params: map[string]any{"name": 99}}, expectCode: mcp.ErrorCodeInvalidParams},
		{name: "resources read error", req: mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "resources/read", ID: 4, Params: map[string]any{"uri": "catalog://items"}}, expectCode: mcp.ErrorCodeInvalidParams},
		{name: "prompts get error", req: mcp.Request{JSONRPC: mcp.JSONRPCVersion, Method: "prompts/get", ID: 5, Params: map[string]any{"name": "promptA"}}, expectCode: mcp.ErrorCodeInvalidParams},
	}

	for _, tc := range tests {
		sender := &captureSender{}
		ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, sender)
		if err := srv.HandleRequest(ctx, tc.req); err != nil {
			t.Fatalf("HandleRequest returned error for %s: %v", tc.name, err)
		}
		if sender.errorCode != tc.expectCode {
			t.Fatalf("%s: expected error code %d, got %d", tc.name, tc.expectCode, sender.errorCode)
		}
	}
}

func TestSendWithoutSenderInContext(t *testing.T) {
	srv, _, _, _ := newServerWithHandlers(t)

	if err := srv.sendResponse(context.Background(), 1, map[string]any{"ok": true}); err == nil {
		t.Fatal("expected error for missing response sender")
	}
	if err := srv.sendError(context.Background(), 1, mcp.ErrorCodeInternalError, "x", nil); err == nil {
		t.Fatal("expected error for missing response sender")
	}
}

func TestSendWithWrongSenderTypeInContext(t *testing.T) {
	srv, _, _, _ := newServerWithHandlers(t)

	ctx := context.WithValue(context.Background(), mcp.ResponseSenderKey, "wrong-type")
	if err := srv.sendResponse(ctx, 1, map[string]any{"ok": true}); err == nil {
		t.Fatal("expected error for wrong response sender type")
	}
	if err := srv.sendError(ctx, 1, mcp.ErrorCodeInternalError, "x", nil); err == nil {
		t.Fatal("expected error for wrong response sender type")
	}
}

func TestParamHelpers(t *testing.T) {
	if _, err := parseParamsMap(nil); err == nil {
		t.Fatal("expected error for nil params")
	}
	if _, err := parseParamsMap("not-map"); err == nil {
		t.Fatal("expected error for non-map params")
	}

	if _, err := requiredStringParam(map[string]any{"name": 10}, "name"); err == nil {
		t.Fatal("expected error for non-string required param")
	}

	args := optionalArguments(map[string]any{"arguments": "not-a-map"})
	if len(args) != 0 {
		t.Fatalf("expected empty arguments for non-map input, got %+v", args)
	}
}
