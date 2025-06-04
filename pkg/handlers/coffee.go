// Package handlers provides domain-specific MCP handler implementations.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// Drink represents a coffee shop drink
type Drink struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Description string `json:"description"`
}

// Coffee implements MCP handlers for a coffee shop domain
type Coffee struct {
	drinks []Drink
}

// NewCoffee creates a new coffee handler with predefined drinks
func NewCoffee() *Coffee {
	return &Coffee{
		drinks: []Drink{
			{
				Name:        "Latte",
				Price:       5,
				Description: "A latte is a coffee drink made with espresso and steamed milk.",
			},
			{
				Name:        "Mocha",
				Price:       6,
				Description: "A mocha is a coffee drink made with espresso and chocolate.",
			},
			{
				Name:        "Flat White",
				Price:       7,
				Description: "A flat white is a coffee drink made with espresso and steamed milk.",
			},
		},
	}
}

// Tool Handler Implementation

func (c *Coffee) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return []mcp.Tool{
		{
			Name:        "getDrinkNames",
			Description: "Get the names of the drinks in the shop",
			InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}},
		},
		{
			Name:        "getDrinkInfo",
			Description: "Get more info about the drink",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]any{
					"name": map[string]string{"type": "string"},
				},
				Required: []string{"name"},
			},
		},
	}, nil
}

func (c *Coffee) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	switch params.Name {
	case "getDrinkNames":
		return c.getDrinkNames(ctx), nil
	case "getDrinkInfo":
		return c.getDrinkInfo(ctx, params.Arguments)
	default:
		return mcp.ToolResponse{}, fmt.Errorf("tool %s not found", params.Name)
	}
}

func (c *Coffee) getDrinkNames(ctx context.Context) mcp.ToolResponse {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return mcp.ToolResponse{
			Content: []mcp.ContentItem{
				{
					Type: "text",
					Text: `{"error": "Request cancelled"}`,
				},
			},
		}
	default:
	}

	var names []string
	for _, drink := range c.drinks {
		names = append(names, drink.Name)
	}

	namesJSON, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		return mcp.ToolResponse{
			Content: []mcp.ContentItem{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "Failed to marshal drink names: %s"}`, err.Error()),
				},
			},
		}
	}

	return mcp.ToolResponse{
		Content: []mcp.ContentItem{
			{
				Type: "text",
				Text: string(namesJSON),
			},
		},
	}
}

func (c *Coffee) getDrinkInfo(ctx context.Context, args map[string]any) (mcp.ToolResponse, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return mcp.ToolResponse{}, ctx.Err()
	default:
	}

	name, ok := args["name"].(string)
	if !ok {
		return mcp.ToolResponse{}, fmt.Errorf("invalid name parameter: expected string")
	}

	for _, drink := range c.drinks {
		if drink.Name == name {
			drinkJSON, err := json.Marshal(drink)
			if err != nil {
				return mcp.ToolResponse{}, fmt.Errorf("failed to marshal drink info: %w", err)
			}
			return mcp.ToolResponse{
				Content: []mcp.ContentItem{
					{
						Type: "text",
						Text: string(drinkJSON),
					},
				},
			}, nil
		}
	}

	return mcp.ToolResponse{}, fmt.Errorf("drink not found: %s", name)
}

// Resource Handler Implementation

func (c *Coffee) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	return []mcp.Resource{
		{
			URI:  "menu://app",
			Name: "menu",
		},
	}, nil
}

func (c *Coffee) ReadResource(ctx context.Context, params mcp.ResourceParams) (mcp.ResourceResponse, error) {
	if params.URI == "menu://app" {
		return c.getMenuResource()
	}
	return mcp.ResourceResponse{}, fmt.Errorf("resource not found: %s", params.URI)
}

func (c *Coffee) getMenuResource() (mcp.ResourceResponse, error) {
	drinksJSON, err := json.Marshal(c.drinks)
	if err != nil {
		return mcp.ResourceResponse{}, fmt.Errorf("failed to marshal menu: %w", err)
	}
	return mcp.ResourceResponse{
		Contents: []mcp.ResourceContent{
			{
				URI:  "menu://app",
				Text: string(drinksJSON),
			},
		},
	}, nil
}

// Prompt Handler Implementation

func (c *Coffee) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	return []mcp.Prompt{
		{
			Name:        "drinkRecommendation",
			Description: "Get personalized drink recommendations based on budget and preferences",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "budget",
					Description: "Customer's budget in dollars",
					Required:    false,
				},
				{
					Name:        "preference",
					Description: "Customer's taste preference (e.g., 'sweet', 'strong', 'mild')",
					Required:    false,
				},
			},
		},
		{
			Name:        "drinkDescription",
			Description: "Get a detailed description and information about a specific coffee drink",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "drink_name",
					Description: "The name of the drink to describe",
					Required:    true,
				},
			},
		},
	}, nil
}

func (c *Coffee) GetPrompt(ctx context.Context, params mcp.PromptParams) (mcp.PromptResponse, error) {
	switch params.Name {
	case "drinkRecommendation":
		return c.createDrinkRecommendationPrompt(params.Arguments), nil
	case "drinkDescription":
		return c.createDrinkDescriptionPrompt(params.Arguments), nil
	default:
		return mcp.PromptResponse{}, fmt.Errorf("prompt %s not found", params.Name)
	}
}

func (c *Coffee) createDrinkRecommendationPrompt(args map[string]any) mcp.PromptResponse {
	budget, hasBudget := args["budget"]
	preference, hasPreference := args["preference"]

	var budgetText string
	if hasBudget {
		budgetText = fmt.Sprintf(" with a budget of $%v", budget)
	}

	var preferenceText string
	if hasPreference {
		preferenceText = fmt.Sprintf(" who prefers %v drinks", preference)
	}

	promptText := fmt.Sprintf(`You are a coffee expert at a specialty coffee shop. A customer%s%s is asking for drink recommendations.

Available drinks:
- Latte ($5): A latte is a coffee drink made with espresso and steamed milk.
- Mocha ($6): A mocha is a coffee drink made with espresso and chocolate.
- Flat White ($7): A flat white is a coffee drink made with espresso and steamed milk.

Please recommend the best drink(s) for this customer and explain why.`, preferenceText, budgetText)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.MessageContent{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}
}

func (c *Coffee) createDrinkDescriptionPrompt(args map[string]any) mcp.PromptResponse {
	drinkName, ok := args["drink_name"].(string)
	if !ok {
		drinkName = "coffee"
	}

	promptText := fmt.Sprintf(`You are a coffee expert. Please provide a detailed description of a %s, including:

1. The ingredients and preparation method
2. The flavor profile and tasting notes
3. The history or origin of this drink
4. Tips for enjoying it

Be engaging and informative in your response.`, drinkName)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.MessageContent{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}
}
