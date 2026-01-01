package mcp

import (
	"encoding/json"
	"testing"
)

func TestTool(t *testing.T) {
	t.Run("create tool with all fields", func(t *testing.T) {
		tool := Tool{
			Name:        "test-tool",
			Description: "A test tool",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"param1": map[string]string{"type": "string"},
				},
				Required: []string{"param1"},
			},
		}

		if tool.Name != "test-tool" {
			t.Errorf("Expected name 'test-tool', got %s", tool.Name)
		}

		if tool.Description != "A test tool" {
			t.Errorf("Expected description 'A test tool', got %s", tool.Description)
		}

		if tool.InputSchema.Type != "object" {
			t.Errorf("Expected input schema type 'object', got %s", tool.InputSchema.Type)
		}

		if len(tool.InputSchema.Properties) != 1 {
			t.Errorf("Expected 1 property, got %d", len(tool.InputSchema.Properties))
		}

		if len(tool.InputSchema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(tool.InputSchema.Required))
		}
	})

	t.Run("create tool with empty schema", func(t *testing.T) {
		tool := Tool{
			Name:        "simple-tool",
			Description: "A simple tool",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]any{},
			},
		}

		if tool.InputSchema.Properties == nil {
			t.Errorf("Expected non-nil properties")
		}

		if len(tool.InputSchema.Required) != 0 {
			t.Errorf("Expected no required fields")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		tool := Tool{
			Name:        "json-tool",
			Description: "Tool for JSON testing",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]any{
					"value": map[string]string{"type": "number"},
				},
				Required: []string{"value"},
			},
		}

		data, err := json.Marshal(tool)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled Tool
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.Name != tool.Name {
			t.Errorf("Expected name %s after round-trip, got %s", tool.Name, unmarshaled.Name)
		}
	})
}

func TestInputSchema(t *testing.T) {
	t.Run("create schema with all fields", func(t *testing.T) {
		schema := InputSchema{
			Type: "object",
			Properties: map[string]any{
				"stringParam": map[string]string{"type": "string"},
				"numberParam": map[string]string{"type": "number"},
			},
			Required: []string{"stringParam"},
		}

		if schema.Type != "object" {
			t.Errorf("Expected type 'object', got %s", schema.Type)
		}

		if len(schema.Properties) != 2 {
			t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
		}

		if len(schema.Required) != 1 {
			t.Errorf("Expected 1 required field, got %d", len(schema.Required))
		}
	})

	t.Run("create schema with only type", func(t *testing.T) {
		schema := InputSchema{
			Type: "string",
		}

		if schema.Properties != nil {
			t.Errorf("Expected nil properties")
		}

		if len(schema.Required) != 0 {
			t.Errorf("Expected no required fields")
		}
	})
}

func TestToolCallParams(t *testing.T) {
	t.Run("create params with all fields", func(t *testing.T) {
		params := ToolCallParams{
			Name: "test-tool",
			Arguments: map[string]any{
				"param1": "value1",
				"param2": 123,
			},
		}

		if params.Name != "test-tool" {
			t.Errorf("Expected name 'test-tool', got %s", params.Name)
		}

		if len(params.Arguments) != 2 {
			t.Errorf("Expected 2 arguments, got %d", len(params.Arguments))
		}

		if params.Arguments["param1"] != "value1" {
			t.Errorf("Expected param1='value1', got %v", params.Arguments["param1"])
		}

		if params.Arguments["param2"] != 123 {
			t.Errorf("Expected param2=123, got %v", params.Arguments["param2"])
		}
	})

	t.Run("create params with empty arguments", func(t *testing.T) {
		params := ToolCallParams{
			Name:      "simple-tool",
			Arguments: map[string]any{},
		}

		if params.Arguments == nil {
			t.Errorf("Expected non-nil arguments")
		}

		if len(params.Arguments) != 0 {
			t.Errorf("Expected no arguments")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		params := ToolCallParams{
			Name: "json-tool",
			Arguments: map[string]any{
				"value": "test",
			},
		}

		data, err := json.Marshal(params)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled ToolCallParams
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.Name != params.Name {
			t.Errorf("Expected name %s after round-trip, got %s", params.Name, unmarshaled.Name)
		}
	})
}

func TestToolResponse(t *testing.T) {
	t.Run("create response with single content item", func(t *testing.T) {
		response := ToolResponse{
			Content: []ContentItem{
				{
					Type: "text",
					Text: "Test response text",
				},
			},
		}

		if len(response.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Content))
		}

		if response.Content[0].Type != "text" {
			t.Errorf("Expected type 'text', got %s", response.Content[0].Type)
		}

		if response.Content[0].Text != "Test response text" {
			t.Errorf("Expected text 'Test response text', got %s", response.Content[0].Text)
		}
	})

	t.Run("create response with multiple content items", func(t *testing.T) {
		response := ToolResponse{
			Content: []ContentItem{
				{Type: "text", Text: "First item"},
				{Type: "text", Text: "Second item"},
				{Type: "text", Text: "Third item"},
			},
		}

		if len(response.Content) != 3 {
			t.Errorf("Expected 3 content items, got %d", len(response.Content))
		}

		if response.Content[0].Text != "First item" {
			t.Errorf("Expected first text 'First item', got %s", response.Content[0].Text)
		}

		if response.Content[2].Text != "Third item" {
			t.Errorf("Expected third text 'Third item', got %s", response.Content[2].Text)
		}
	})

	t.Run("create response with empty content", func(t *testing.T) {
		response := ToolResponse{
			Content: []ContentItem{},
		}

		if response.Content == nil {
			t.Errorf("Expected non-nil content")
		}

		if len(response.Content) != 0 {
			t.Errorf("Expected no content items")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		response := ToolResponse{
			Content: []ContentItem{
				{Type: "text", Text: "Test content"},
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled ToolResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if len(unmarshaled.Content) != 1 {
			t.Errorf("Expected 1 content item after round-trip, got %d", len(unmarshaled.Content))
		}
	})
}

func TestContentItem(t *testing.T) {
	t.Run("create text content item", func(t *testing.T) {
		item := ContentItem{
			Type: "text",
			Text: "Sample text content",
		}

		if item.Type != "text" {
			t.Errorf("Expected type 'text', got %s", item.Type)
		}

		if item.Text != "Sample text content" {
			t.Errorf("Expected text 'Sample text content', got %s", item.Text)
		}
	})

	t.Run("create content item with empty text", func(t *testing.T) {
		item := ContentItem{
			Type: "text",
			Text: "",
		}

		if item.Text != "" {
			t.Errorf("Expected empty text, got '%s'", item.Text)
		}
	})

	t.Run("create content item with multiline text", func(t *testing.T) {
		item := ContentItem{
			Type: "text",
			Text: "Line 1\nLine 2\nLine 3",
		}

		if !contains(item.Text, "\n") {
			t.Errorf("Expected text to contain newlines")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)*2 && containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
