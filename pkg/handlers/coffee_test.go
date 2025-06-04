package handlers

import (
	"context"
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
}
