package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
	"github.com/BearHuddleston/mcp-server-example/pkg/spec"
)

func TestCatalogHandler(t *testing.T) {
	handler := NewCatalog()
	ctx := context.Background()

	t.Run("ListTools", func(t *testing.T) {
		tools, err := handler.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}

		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}

		expectedTools := map[string]bool{
			"listItems":      false,
			"getItemDetails": false,
		}

		for _, tool := range tools {
			if _, exists := expectedTools[tool.Name]; exists {
				expectedTools[tool.Name] = true
			}
		}

		for name, found := range expectedTools {
			if !found {
				t.Errorf("Expected tool %s not found", name)
			}
		}
	})

	t.Run("CallTool - listItems", func(t *testing.T) {
		params := mcp.ToolCallParams{Name: "listItems", Arguments: map[string]any{}}

		response, err := handler.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if len(response.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Content))
		}

		if response.Content[0].Type != "text" {
			t.Errorf("Expected content type 'text', got %s", response.Content[0].Type)
		}
	})

	t.Run("CallTool - getItemDetails success", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getItemDetails",
			Arguments: map[string]any{"name": "Workspace Automation Pack"},
		}

		response, err := handler.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
		if len(response.Content) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(response.Content))
		}
		if !strings.Contains(response.Content[0].Text, "Workspace Automation Pack") {
			t.Fatalf("Expected response to include item name, got %s", response.Content[0].Text)
		}
	})

	t.Run("CallTool - unknown tool", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "unknown", Arguments: map[string]any{}})
		if err == nil {
			t.Fatal("Expected error for unknown tool")
		}
	})

	t.Run("CallTool - invalid getItemDetails args", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "getItemDetails", Arguments: map[string]any{"name": 42}})
		if err == nil {
			t.Fatal("Expected error for invalid name argument")
		}
	})

	t.Run("CallTool - missing item", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "getItemDetails", Arguments: map[string]any{"name": "Unknown"}})
		if err == nil {
			t.Fatal("Expected error for unknown item")
		}
	})

	t.Run("CallTool - cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := handler.CallTool(cancelledCtx, mcp.ToolCallParams{Name: "listItems", Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("Expected no error for listItems cancellation branch, got %v", err)
		}
		if len(resp.Content) != 1 || !strings.Contains(resp.Content[0].Text, "request cancelled") {
			t.Fatalf("Expected cancellation response content, got %+v", resp.Content)
		}

		_, err = handler.CallTool(cancelledCtx, mcp.ToolCallParams{Name: "getItemDetails", Arguments: map[string]any{"name": "Workspace Automation Pack"}})
		if err == nil {
			t.Fatal("Expected context cancellation error for getItemDetails")
		}
	})

	t.Run("ListResources", func(t *testing.T) {
		resources, err := handler.ListResources(ctx)
		if err != nil {
			t.Fatalf("ListResources failed: %v", err)
		}

		if len(resources) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(resources))
		}

		if resources[0].URI != "catalog://items" {
			t.Errorf("Expected URI 'catalog://items', got %s", resources[0].URI)
		}
	})

	t.Run("ReadResource success", func(t *testing.T) {
		resp, err := handler.ReadResource(ctx, mcp.ResourceParams{URI: "catalog://items"})
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}
		if len(resp.Contents) != 1 {
			t.Fatalf("Expected 1 resource content, got %d", len(resp.Contents))
		}
	})

	t.Run("ReadResource unknown", func(t *testing.T) {
		_, err := handler.ReadResource(ctx, mcp.ResourceParams{URI: "missing://items"})
		if err == nil {
			t.Fatal("Expected error for unknown resource")
		}
	})

	t.Run("ListPrompts and GetPrompt", func(t *testing.T) {
		prompts, err := handler.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("ListPrompts failed: %v", err)
		}
		if len(prompts) != 2 {
			t.Fatalf("Expected 2 prompts, got %d", len(prompts))
		}

		recommendation, err := handler.GetPrompt(ctx, mcp.PromptParams{
			Name:      "planRecommendation",
			Arguments: map[string]any{"budget": 6, "goal": "speed"},
		})
		if err != nil {
			t.Fatalf("GetPrompt planRecommendation failed: %v", err)
		}
		if len(recommendation.Messages) != 1 || !strings.Contains(recommendation.Messages[0].Content.Text, "speed") {
			t.Fatalf("Expected recommendation text to include goal, got %+v", recommendation.Messages)
		}

		description, err := handler.GetPrompt(ctx, mcp.PromptParams{
			Name:      "itemBrief",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("GetPrompt itemBrief failed: %v", err)
		}
		if len(description.Messages) != 1 || !strings.Contains(description.Messages[0].Content.Text, "catalog item") {
			t.Fatalf("Expected default item name in prompt, got %+v", description.Messages)
		}

		_, err = handler.GetPrompt(ctx, mcp.PromptParams{Name: "unknown", Arguments: map[string]any{}})
		if err == nil {
			t.Fatal("Expected error for unknown prompt")
		}
	})
}

func TestNewCatalogFromSpec(t *testing.T) {
	t.Run("nil spec", func(t *testing.T) {
		_, err := NewCatalogFromSpec(nil)
		if err == nil {
			t.Fatal("expected error for nil spec")
		}
	})

	t.Run("maps tools resources and prompts", func(t *testing.T) {
		sp := &spec.Spec{
			SchemaVersion: "v1",
			Items: []spec.ItemSpec{{
				"item_name": "Template Bundle",
				"score":     9,
				"track":     "starter",
			}},
			Tools: []spec.ToolSpec{
				{
					Mode:        "list_items",
					Name:        "listCatalog",
					Description: "List names from catalog",
					InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}, Required: []string{}},
				},
				{
					Mode:        "get_item_details",
					Name:        "fetchDetails",
					Description: "Get details for selected item",
					InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{"item_name": map[string]string{"type": "string"}}, Required: []string{"item_name"}},
				},
			},
			Resources: []spec.ResourceSpec{{Mode: "catalog_items", URI: "catalog://custom-items", Name: "custom-catalog"}},
			Prompts: []spec.PromptSpec{
				{
					Mode:        "plan_recommendation",
					Name:        "buildPlan",
					Description: "Build a recommendation",
					Arguments: []mcp.PromptArgument{
						{Name: "budget", Description: "Max budget", Required: false},
						{Name: "goal", Description: "Primary goal", Required: false},
					},
					Template: "Plan for a team%s%s",
				},
				{
					Mode:        "item_brief",
					Name:        "quickBrief",
					Description: "Write item brief",
					Arguments:   []mcp.PromptArgument{{Name: "target_item", Description: "Item to summarize", Required: true}},
					Template:    "Brief for %s",
				},
			},
		}

		h, err := NewCatalogFromSpec(sp)
		if err != nil {
			t.Fatalf("NewCatalogFromSpec failed: %v", err)
		}

		ctx := context.Background()
		tools, err := h.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}
		if len(tools) != 2 {
			t.Fatalf("unexpected tools: %+v", tools)
		}
		found := map[string]bool{"listCatalog": false, "fetchDetails": false}
		for _, tool := range tools {
			if _, ok := found[tool.Name]; ok {
				found[tool.Name] = true
			}
		}
		for name, ok := range found {
			if !ok {
				t.Fatalf("expected tool %q not found in %+v", name, tools)
			}
		}

		toolResp, err := h.CallTool(ctx, mcp.ToolCallParams{Name: "fetchDetails", Arguments: map[string]any{"item_name": "Template Bundle"}})
		if err != nil {
			t.Fatalf("CallTool fetchDetails failed: %v", err)
		}
		if len(toolResp.Content) != 1 || !strings.Contains(toolResp.Content[0].Text, "Template Bundle") {
			t.Fatalf("unexpected tool response: %+v", toolResp.Content)
		}

		resources, err := h.ListResources(ctx)
		if err != nil {
			t.Fatalf("ListResources failed: %v", err)
		}
		if len(resources) != 1 || resources[0].URI != "catalog://custom-items" {
			t.Fatalf("unexpected resources: %+v", resources)
		}

		promptResp, err := h.GetPrompt(ctx, mcp.PromptParams{Name: "quickBrief", Arguments: map[string]any{"target_item": "Template Bundle"}})
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}
		if len(promptResp.Messages) != 1 || !strings.Contains(promptResp.Messages[0].Content.Text, "Template Bundle") {
			t.Fatalf("unexpected prompt response: %+v", promptResp.Messages)
		}
	})
}
