package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

func TestCoffeeHandler(t *testing.T) {
	handler := NewCoffee()
	ctx := context.Background()

	t.Run("ListTools", func(t *testing.T) {
		tools, err := handler.ListTools(ctx)
		if err != nil {
			t.Fatalf("ListTools failed: %v", err)
		}

		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}

		// Verify tool names
		expectedTools := map[string]bool{
			"getDrinkNames": false,
			"getDrinkInfo":  false,
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

	t.Run("CallTool - getDrinkNames", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkNames",
			Arguments: map[string]any{},
		}

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

	t.Run("CallTool - getDrinkInfo success", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{"name": "Latte"},
		}

		response, err := handler.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
		if len(response.Content) != 1 {
			t.Fatalf("Expected 1 content item, got %d", len(response.Content))
		}
		if !strings.Contains(response.Content[0].Text, "Latte") {
			t.Fatalf("Expected response to include Latte, got %s", response.Content[0].Text)
		}
	})

	t.Run("CallTool - unknown tool", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "unknown", Arguments: map[string]any{}})
		if err == nil {
			t.Fatal("Expected error for unknown tool")
		}
	})

	t.Run("CallTool - invalid getDrinkInfo args", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "getDrinkInfo", Arguments: map[string]any{"name": 42}})
		if err == nil {
			t.Fatal("Expected error for invalid name argument")
		}
	})

	t.Run("CallTool - missing drink", func(t *testing.T) {
		_, err := handler.CallTool(ctx, mcp.ToolCallParams{Name: "getDrinkInfo", Arguments: map[string]any{"name": "Unknown"}})
		if err == nil {
			t.Fatal("Expected error for unknown drink")
		}
	})

	t.Run("CallTool - cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		resp, err := handler.CallTool(cancelledCtx, mcp.ToolCallParams{Name: "getDrinkNames", Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("Expected no error for getDrinkNames cancellation branch, got %v", err)
		}
		if len(resp.Content) != 1 || !strings.Contains(resp.Content[0].Text, "Request cancelled") {
			t.Fatalf("Expected cancellation response content, got %+v", resp.Content)
		}

		_, err = handler.CallTool(cancelledCtx, mcp.ToolCallParams{Name: "getDrinkInfo", Arguments: map[string]any{"name": "Latte"}})
		if err == nil {
			t.Fatal("Expected context cancellation error for getDrinkInfo")
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

		if resources[0].URI != "menu://app" {
			t.Errorf("Expected URI 'menu://app', got %s", resources[0].URI)
		}
	})

	t.Run("ReadResource success", func(t *testing.T) {
		resp, err := handler.ReadResource(ctx, mcp.ResourceParams{URI: "menu://app"})
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}
		if len(resp.Contents) != 1 {
			t.Fatalf("Expected 1 resource content, got %d", len(resp.Contents))
		}
	})

	t.Run("ReadResource unknown", func(t *testing.T) {
		_, err := handler.ReadResource(ctx, mcp.ResourceParams{URI: "missing://app"})
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
			Name:      "drinkRecommendation",
			Arguments: map[string]any{"budget": 6, "preference": "sweet"},
		})
		if err != nil {
			t.Fatalf("GetPrompt drinkRecommendation failed: %v", err)
		}
		if len(recommendation.Messages) != 1 || !strings.Contains(recommendation.Messages[0].Content.Text, "sweet") {
			t.Fatalf("Expected recommendation text to include preference, got %+v", recommendation.Messages)
		}

		description, err := handler.GetPrompt(ctx, mcp.PromptParams{
			Name:      "drinkDescription",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("GetPrompt drinkDescription failed: %v", err)
		}
		if len(description.Messages) != 1 || !strings.Contains(description.Messages[0].Content.Text, "coffee") {
			t.Fatalf("Expected default drink name in description prompt, got %+v", description.Messages)
		}

		_, err = handler.GetPrompt(ctx, mcp.PromptParams{Name: "unknown", Arguments: map[string]any{}})
		if err == nil {
			t.Fatal("Expected error for unknown prompt")
		}
	})
}
