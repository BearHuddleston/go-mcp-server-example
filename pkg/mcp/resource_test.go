package mcp

import (
	"encoding/json"
	"testing"
)

func TestResource(t *testing.T) {
	t.Run("create resource with all fields", func(t *testing.T) {
		resource := Resource{
			URI:  "test://resource/123",
			Name: "Test Resource",
		}

		if resource.URI != "test://resource/123" {
			t.Errorf("Expected URI 'test://resource/123', got %s", resource.URI)
		}

		if resource.Name != "Test Resource" {
			t.Errorf("Expected name 'Test Resource', got %s", resource.Name)
		}
	})

	t.Run("create resource with empty name", func(t *testing.T) {
		resource := Resource{
			URI:  "test://resource/456",
			Name: "",
		}

		if resource.Name != "" {
			t.Errorf("Expected empty name, got '%s'", resource.Name)
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		resource := Resource{
			URI:  "json://test",
			Name: "JSON Test Resource",
		}

		data, err := json.Marshal(resource)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled Resource
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.URI != resource.URI {
			t.Errorf("Expected URI %s after round-trip, got %s", resource.URI, unmarshaled.URI)
		}

		if unmarshaled.Name != resource.Name {
			t.Errorf("Expected name %s after round-trip, got %s", resource.Name, unmarshaled.Name)
		}
	})
}

func TestResourceContent(t *testing.T) {
	t.Run("create resource content with all fields", func(t *testing.T) {
		content := ResourceContent{
			URI:  "test://resource/789",
			Text: "Resource content text",
		}

		if content.URI != "test://resource/789" {
			t.Errorf("Expected URI 'test://resource/789', got %s", content.URI)
		}

		if content.Text != "Resource content text" {
			t.Errorf("Expected text 'Resource content text', got %s", content.Text)
		}
	})

	t.Run("create resource content with empty text", func(t *testing.T) {
		content := ResourceContent{
			URI:  "test://resource/empty",
			Text: "",
		}

		if content.Text != "" {
			t.Errorf("Expected empty text, got '%s'", content.Text)
		}
	})

	t.Run("create resource content with multiline text", func(t *testing.T) {
		content := ResourceContent{
			URI:  "test://resource/multiline",
			Text: "Line 1\nLine 2\nLine 3",
		}

		if !containsNewline(content.Text) {
			t.Errorf("Expected text to contain newlines")
		}
	})

	t.Run("create resource content with JSON text", func(t *testing.T) {
		jsonText := `{"key": "value", "number": 123}`
		content := ResourceContent{
			URI:  "test://resource/json",
			Text: jsonText,
		}

		if content.Text != jsonText {
			t.Errorf("Expected JSON text '%s', got '%s'", jsonText, content.Text)
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		content := ResourceContent{
			URI:  "json://content",
			Text: "Test content",
		}

		data, err := json.Marshal(content)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled ResourceContent
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.URI != content.URI {
			t.Errorf("Expected URI %s after round-trip, got %s", content.URI, unmarshaled.URI)
		}

		if unmarshaled.Text != content.Text {
			t.Errorf("Expected text %s after round-trip, got %s", content.Text, unmarshaled.Text)
		}
	})
}

func TestResourceResponse(t *testing.T) {
	t.Run("create response with single content item", func(t *testing.T) {
		response := ResourceResponse{
			Contents: []ResourceContent{
				{
					URI:  "test://resource/1",
					Text: "First resource content",
				},
			},
		}

		if len(response.Contents) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(response.Contents))
		}

		if response.Contents[0].URI != "test://resource/1" {
			t.Errorf("Expected URI 'test://resource/1', got %s", response.Contents[0].URI)
		}

		if response.Contents[0].Text != "First resource content" {
			t.Errorf("Expected text 'First resource content', got %s", response.Contents[0].Text)
		}
	})

	t.Run("create response with multiple content items", func(t *testing.T) {
		response := ResourceResponse{
			Contents: []ResourceContent{
				{URI: "test://resource/1", Text: "Content 1"},
				{URI: "test://resource/2", Text: "Content 2"},
				{URI: "test://resource/3", Text: "Content 3"},
			},
		}

		if len(response.Contents) != 3 {
			t.Errorf("Expected 3 content items, got %d", len(response.Contents))
		}

		if response.Contents[0].Text != "Content 1" {
			t.Errorf("Expected first text 'Content 1', got %s", response.Contents[0].Text)
		}

		if response.Contents[1].Text != "Content 2" {
			t.Errorf("Expected second text 'Content 2', got %s", response.Contents[1].Text)
		}

		if response.Contents[2].Text != "Content 3" {
			t.Errorf("Expected third text 'Content 3', got %s", response.Contents[2].Text)
		}
	})

	t.Run("create response with empty contents", func(t *testing.T) {
		response := ResourceResponse{
			Contents: []ResourceContent{},
		}

		if response.Contents == nil {
			t.Errorf("Expected non-nil contents")
		}

		if len(response.Contents) != 0 {
			t.Errorf("Expected no content items")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		response := ResourceResponse{
			Contents: []ResourceContent{
				{URI: "json://content/1", Text: "Content 1"},
				{URI: "json://content/2", Text: "Content 2"},
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled ResourceResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if len(unmarshaled.Contents) != 2 {
			t.Errorf("Expected 2 content items after round-trip, got %d", len(unmarshaled.Contents))
		}
	})
}

func TestResourceParams(t *testing.T) {
	t.Run("create params with URI", func(t *testing.T) {
		params := ResourceParams{
			URI: "test://resource/123",
		}

		if params.URI != "test://resource/123" {
			t.Errorf("Expected URI 'test://resource/123', got %s", params.URI)
		}
	})

	t.Run("create params with empty URI", func(t *testing.T) {
		params := ResourceParams{
			URI: "",
		}

		if params.URI != "" {
			t.Errorf("Expected empty URI, got '%s'", params.URI)
		}
	})

	t.Run("create params with file URI", func(t *testing.T) {
		params := ResourceParams{
			URI: "file:///path/to/file.txt",
		}

		if params.URI != "file:///path/to/file.txt" {
			t.Errorf("Expected URI 'file:///path/to/file.txt', got %s", params.URI)
		}
	})

	t.Run("create params with http URI", func(t *testing.T) {
		params := ResourceParams{
			URI: "https://example.com/resource",
		}

		if params.URI != "https://example.com/resource" {
			t.Errorf("Expected URI 'https://example.com/resource', got %s", params.URI)
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		params := ResourceParams{
			URI: "json://params",
		}

		data, err := json.Marshal(params)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled ResourceParams
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.URI != params.URI {
			t.Errorf("Expected URI %s after round-trip, got %s", params.URI, unmarshaled.URI)
		}
	})
}

func containsNewline(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return true
		}
	}
	return false
}
