package mcp

import (
	"encoding/json"
	"testing"
)

func TestPrompt(t *testing.T) {
	t.Run("create prompt with all fields", func(t *testing.T) {
		prompt := Prompt{
			Name:        "test-prompt",
			Description: "A test prompt for testing",
			Arguments: []PromptArgument{
				{
					Name:        "arg1",
					Description: "First argument",
					Required:    true,
				},
				{
					Name:        "arg2",
					Description: "Second argument",
					Required:    false,
				},
			},
		}

		if prompt.Name != "test-prompt" {
			t.Errorf("Expected name 'test-prompt', got %s", prompt.Name)
		}

		if prompt.Description != "A test prompt for testing" {
			t.Errorf("Expected description 'A test prompt for testing', got %s", prompt.Description)
		}

		if len(prompt.Arguments) != 2 {
			t.Errorf("Expected 2 arguments, got %d", len(prompt.Arguments))
		}

		if !prompt.Arguments[0].Required {
			t.Errorf("Expected first argument to be required")
		}

		if prompt.Arguments[1].Required {
			t.Errorf("Expected second argument to be optional")
		}
	})

	t.Run("create prompt without arguments", func(t *testing.T) {
		prompt := Prompt{
			Name:        "simple-prompt",
			Description: "A simple prompt with no arguments",
		}

		if len(prompt.Arguments) != 0 {
			t.Errorf("Expected no arguments, got %d", len(prompt.Arguments))
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		prompt := Prompt{
			Name:        "json-prompt",
			Description: "JSON test prompt",
			Arguments: []PromptArgument{
				{Name: "param", Description: "Test parameter", Required: false},
			},
		}

		data, err := json.Marshal(prompt)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled Prompt
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.Name != prompt.Name {
			t.Errorf("Expected name %s after round-trip, got %s", prompt.Name, unmarshaled.Name)
		}
	})
}

func TestPromptArgument(t *testing.T) {
	t.Run("create argument with all fields", func(t *testing.T) {
		arg := PromptArgument{
			Name:        "test-arg",
			Description: "A test argument",
			Required:    true,
		}

		if arg.Name != "test-arg" {
			t.Errorf("Expected name 'test-arg', got %s", arg.Name)
		}

		if arg.Description != "A test argument" {
			t.Errorf("Expected description 'A test argument', got %s", arg.Description)
		}

		if !arg.Required {
			t.Errorf("Expected argument to be required")
		}
	})

	t.Run("create optional argument", func(t *testing.T) {
		arg := PromptArgument{
			Name:        "optional-arg",
			Description: "An optional argument",
			Required:    false,
		}

		if arg.Required {
			t.Errorf("Expected argument to be optional")
		}
	})

	t.Run("create argument with empty description", func(t *testing.T) {
		arg := PromptArgument{
			Name:        "minimal-arg",
			Description: "",
			Required:    true,
		}

		if arg.Description != "" {
			t.Errorf("Expected empty description, got '%s'", arg.Description)
		}
	})
}

func TestPromptParams(t *testing.T) {
	t.Run("create params with all fields", func(t *testing.T) {
		params := PromptParams{
			Name: "test-prompt",
			Arguments: map[string]any{
				"arg1": "value1",
				"arg2": 123,
				"arg3": true,
			},
		}

		if params.Name != "test-prompt" {
			t.Errorf("Expected name 'test-prompt', got %s", params.Name)
		}

		if len(params.Arguments) != 3 {
			t.Errorf("Expected 3 arguments, got %d", len(params.Arguments))
		}

		if params.Arguments["arg1"] != "value1" {
			t.Errorf("Expected arg1='value1', got %v", params.Arguments["arg1"])
		}

		if params.Arguments["arg2"] != 123 {
			t.Errorf("Expected arg2=123, got %v", params.Arguments["arg2"])
		}

		if params.Arguments["arg3"] != true {
			t.Errorf("Expected arg3=true, got %v", params.Arguments["arg3"])
		}
	})

	t.Run("create params with empty arguments", func(t *testing.T) {
		params := PromptParams{
			Name:      "simple-prompt",
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
		params := PromptParams{
			Name: "json-prompt",
			Arguments: map[string]any{
				"value": "test",
			},
		}

		data, err := json.Marshal(params)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled PromptParams
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if unmarshaled.Name != params.Name {
			t.Errorf("Expected name %s after round-trip, got %s", params.Name, unmarshaled.Name)
		}
	})
}

func TestPromptResponse(t *testing.T) {
	t.Run("create response with single message", func(t *testing.T) {
		response := PromptResponse{
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: MessageContent{
						Type: "text",
						Text: "Test message",
					},
				},
			},
		}

		if len(response.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(response.Messages))
		}

		if response.Messages[0].Role != "user" {
			t.Errorf("Expected role 'user', got %s", response.Messages[0].Role)
		}

		if response.Messages[0].Content.Type != "text" {
			t.Errorf("Expected content type 'text', got %s", response.Messages[0].Content.Type)
		}

		if response.Messages[0].Content.Text != "Test message" {
			t.Errorf("Expected text 'Test message', got %s", response.Messages[0].Content.Text)
		}
	})

	t.Run("create response with multiple messages", func(t *testing.T) {
		response := PromptResponse{
			Messages: []PromptMessage{
				{
					Role:    "system",
					Content: MessageContent{Type: "text", Text: "System instruction"},
				},
				{
					Role:    "user",
					Content: MessageContent{Type: "text", Text: "User message"},
				},
				{
					Role:    "assistant",
					Content: MessageContent{Type: "text", Text: "Assistant response"},
				},
			},
		}

		if len(response.Messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(response.Messages))
		}

		if response.Messages[0].Role != "system" {
			t.Errorf("Expected first role 'system', got %s", response.Messages[0].Role)
		}

		if response.Messages[1].Role != "user" {
			t.Errorf("Expected second role 'user', got %s", response.Messages[1].Role)
		}

		if response.Messages[2].Role != "assistant" {
			t.Errorf("Expected third role 'assistant', got %s", response.Messages[2].Role)
		}
	})

	t.Run("create response with empty messages", func(t *testing.T) {
		response := PromptResponse{
			Messages: []PromptMessage{},
		}

		if response.Messages == nil {
			t.Errorf("Expected non-nil messages")
		}

		if len(response.Messages) != 0 {
			t.Errorf("Expected no messages")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		response := PromptResponse{
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: MessageContent{Type: "text", Text: "Test message"},
				},
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Expected no error marshaling, got: %v", err)
		}

		var unmarshaled PromptResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Errorf("Expected no error unmarshaling, got: %v", err)
		}

		if len(unmarshaled.Messages) != 1 {
			t.Errorf("Expected 1 message after round-trip, got %d", len(unmarshaled.Messages))
		}
	})
}

func TestPromptMessage(t *testing.T) {
	t.Run("create user message", func(t *testing.T) {
		message := PromptMessage{
			Role: "user",
			Content: MessageContent{
				Type: "text",
				Text: "User's message",
			},
		}

		if message.Role != "user" {
			t.Errorf("Expected role 'user', got %s", message.Role)
		}

		if message.Content.Text != "User's message" {
			t.Errorf("Expected text \"User's message\", got %s", message.Content.Text)
		}
	})

	t.Run("create system message", func(t *testing.T) {
		message := PromptMessage{
			Role: "system",
			Content: MessageContent{
				Type: "text",
				Text: "System instruction",
			},
		}

		if message.Role != "system" {
			t.Errorf("Expected role 'system', got %s", message.Role)
		}
	})

	t.Run("create assistant message", func(t *testing.T) {
		message := PromptMessage{
			Role: "assistant",
			Content: MessageContent{
				Type: "text",
				Text: "Assistant's response",
			},
		}

		if message.Role != "assistant" {
			t.Errorf("Expected role 'assistant', got %s", message.Role)
		}
	})
}

func TestMessageContent(t *testing.T) {
	t.Run("create text content", func(t *testing.T) {
		content := MessageContent{
			Type: "text",
			Text: "Sample text content",
		}

		if content.Type != "text" {
			t.Errorf("Expected type 'text', got %s", content.Type)
		}

		if content.Text != "Sample text content" {
			t.Errorf("Expected text 'Sample text content', got %s", content.Text)
		}
	})

	t.Run("create content with empty text", func(t *testing.T) {
		content := MessageContent{
			Type: "text",
			Text: "",
		}

		if content.Text != "" {
			t.Errorf("Expected empty text, got '%s'", content.Text)
		}
	})

	t.Run("create content with multiline text", func(t *testing.T) {
		content := MessageContent{
			Type: "text",
			Text: "Line 1\nLine 2\nLine 3",
		}

		if !containsNewline(content.Text) {
			t.Errorf("Expected text to contain newlines")
		}
	})

	t.Run("create content with long text", func(t *testing.T) {
		longText := "This is a very long text that exceeds normal length. " +
			"It contains multiple sentences and paragraphs. " +
			"The purpose is to test that the MessageContent structure " +
			"can handle text content of significant size without any issues."

		content := MessageContent{
			Type: "text",
			Text: longText,
		}

		if len(content.Text) != len(longText) {
			t.Errorf("Expected text length %d, got %d", len(longText), len(content.Text))
		}
	})
}
