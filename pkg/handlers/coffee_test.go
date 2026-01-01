package handlers

import (
	"context"
	"encoding/json"
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

	t.Run("ReadResource - success", func(t *testing.T) {
		params := mcp.ResourceParams{
			URI: "menu://app",
		}

		response, err := handler.ReadResource(ctx, params)
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}

		if len(response.Contents) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Contents))
		}

		if response.Contents[0].URI != "menu://app" {
			t.Errorf("Expected URI 'menu://app', got %s", response.Contents[0].URI)
		}

		// Verify JSON structure
		var drinks []Drink
		if err := json.Unmarshal([]byte(response.Contents[0].Text), &drinks); err != nil {
			t.Errorf("Failed to parse drinks JSON: %v", err)
		}

		if len(drinks) != 3 {
			t.Errorf("Expected 3 drinks, got %d", len(drinks))
		}
	})

	t.Run("ReadResource - not found", func(t *testing.T) {
		params := mcp.ResourceParams{
			URI: "unknown://resource",
		}

		_, err := handler.ReadResource(ctx, params)
		if err == nil {
			t.Errorf("Expected error for unknown resource, got nil")
		}

		if !strings.Contains(err.Error(), "resource not found") {
			t.Errorf("Expected 'resource not found' error, got: %v", err)
		}
	})

	t.Run("ListPrompts", func(t *testing.T) {
		prompts, err := handler.ListPrompts(ctx)
		if err != nil {
			t.Fatalf("ListPrompts failed: %v", err)
		}

		if len(prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(prompts))
		}

		// Verify prompt names
		expectedPrompts := map[string]bool{
			"drinkRecommendation": false,
			"drinkDescription":    false,
		}

		for _, prompt := range prompts {
			if _, exists := expectedPrompts[prompt.Name]; exists {
				expectedPrompts[prompt.Name] = true
			}
		}

		for name, found := range expectedPrompts {
			if !found {
				t.Errorf("Expected prompt %s not found", name)
			}
		}
	})

	t.Run("GetPrompt - drinkRecommendation", func(t *testing.T) {
		params := mcp.PromptParams{
			Name: "drinkRecommendation",
			Arguments: map[string]any{
				"budget":     "10",
				"preference": "sweet",
			},
		}

		response, err := handler.GetPrompt(ctx, params)
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if len(response.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(response.Messages))
		}

		if response.Messages[0].Role != "user" {
			t.Errorf("Expected role 'user', got %s", response.Messages[0].Role)
		}

		if !strings.Contains(response.Messages[0].Content.Text, "coffee expert") {
			t.Errorf("Expected prompt text to contain 'coffee expert', got: %s", response.Messages[0].Content.Text)
		}

		if !strings.Contains(response.Messages[0].Content.Text, "budget of $10") {
			t.Errorf("Expected prompt text to contain budget, got: %s", response.Messages[0].Content.Text)
		}

		if !strings.Contains(response.Messages[0].Content.Text, "prefers sweet drinks") {
			t.Errorf("Expected prompt text to contain preference, got: %s", response.Messages[0].Content.Text)
		}
	})

	t.Run("GetPrompt - drinkDescription", func(t *testing.T) {
		params := mcp.PromptParams{
			Name: "drinkDescription",
			Arguments: map[string]any{
				"drink_name": "Latte",
			},
		}

		response, err := handler.GetPrompt(ctx, params)
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if len(response.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(response.Messages))
		}

		if response.Messages[0].Role != "user" {
			t.Errorf("Expected role 'user', got %s", response.Messages[0].Role)
		}

		if !strings.Contains(response.Messages[0].Content.Text, "Latte") {
			t.Errorf("Expected prompt text to contain 'Latte', got: %s", response.Messages[0].Content.Text)
		}
	})

	t.Run("GetPrompt - drinkDescription without name", func(t *testing.T) {
		params := mcp.PromptParams{
			Name:      "drinkDescription",
			Arguments: map[string]any{},
		}

		response, err := handler.GetPrompt(ctx, params)
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if !strings.Contains(response.Messages[0].Content.Text, "coffee") {
			t.Errorf("Expected prompt text to default to 'coffee', got: %s", response.Messages[0].Content.Text)
		}
	})

	// Error scenarios
	t.Run("CallTool - unknown tool", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "unknown-tool",
			Arguments: map[string]any{},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected error for unknown tool, got nil")
		}

		if !strings.Contains(err.Error(), "tool unknown-tool not found") {
			t.Errorf("Expected 'tool not found' error, got: %v", err)
		}
	})

	t.Run("CallTool - getDrinkInfo - missing name parameter", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected error for missing name parameter, got nil")
		}

		if !strings.Contains(err.Error(), "invalid name parameter") {
			t.Errorf("Expected 'invalid name parameter' error, got: %v", err)
		}
	})

	t.Run("CallTool - getDrinkInfo - invalid name parameter type", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{"name": 123},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected error for invalid name parameter type, got nil")
		}

		if !strings.Contains(err.Error(), "invalid name parameter") {
			t.Errorf("Expected 'invalid name parameter' error, got: %v", err)
		}
	})

	t.Run("CallTool - getDrinkInfo - drink not found", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{"name": "Nonexistent Drink"},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected error for nonexistent drink, got nil")
		}

		if !strings.Contains(err.Error(), "drink not found") {
			t.Errorf("Expected 'drink not found' error, got: %v", err)
		}

		if !strings.Contains(err.Error(), "Nonexistent Drink") {
			t.Errorf("Expected error to contain drink name, got: %v", err)
		}
	})

	t.Run("GetPrompt - unknown prompt", func(t *testing.T) {
		params := mcp.PromptParams{
			Name:      "unknown-prompt",
			Arguments: map[string]any{},
		}

		_, err := handler.GetPrompt(ctx, params)
		if err == nil {
			t.Errorf("Expected error for unknown prompt, got nil")
		}

		if !strings.Contains(err.Error(), "prompt unknown-prompt not found") {
			t.Errorf("Expected 'prompt not found' error, got: %v", err)
		}
	})

	// Context cancellation tests
	t.Run("getDrinkNames - context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		params := mcp.ToolCallParams{
			Name:      "getDrinkNames",
			Arguments: map[string]any{},
		}

		response, err := handler.CallTool(ctx, params)
		if err != nil {
			t.Errorf("getDrinkNames should handle cancellation gracefully, got error: %v", err)
		}

		if len(response.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Content))
		}

		if !strings.Contains(response.Content[0].Text, "error") {
			t.Errorf("Expected error message in response, got: %s", response.Content[0].Text)
		}
	})

	t.Run("getDrinkInfo - context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{"name": "Latte"},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected context canceled error, got nil")
		}

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	// Edge cases
	t.Run("getDrinkInfo - empty name string", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "getDrinkInfo",
			Arguments: map[string]any{"name": ""},
		}

		_, err := handler.CallTool(ctx, params)
		if err == nil {
			t.Errorf("Expected error for empty name, got nil")
		}

		if !strings.Contains(err.Error(), "drink not found") {
			t.Errorf("Expected 'drink not found' error, got: %v", err)
		}
	})

	t.Run("CallTool - nil arguments", func(t *testing.T) {
		// This tests that nil arguments are handled gracefully
		params := mcp.ToolCallParams{
			Name:      "getDrinkNames",
			Arguments: nil,
		}

		response, err := handler.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("CallTool with nil arguments failed: %v", err)
		}

		if len(response.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Content))
		}

		if response.Content[0].Type != "text" {
			t.Errorf("Expected content type 'text', got %s", response.Content[0].Type)
		}
	})

	t.Run("GetPrompt - nil arguments", func(t *testing.T) {
		params := mcp.PromptParams{
			Name:      "drinkRecommendation",
			Arguments: nil,
		}

		response, err := handler.GetPrompt(ctx, params)
		if err != nil {
			t.Fatalf("GetPrompt with nil arguments failed: %v", err)
		}

		if len(response.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(response.Messages))
		}
	})
}
