package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

type Item struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type Catalog struct {
	items []Item
}

func NewCatalog() *Catalog {
	return &Catalog{
		items: []Item{
			{
				Name:        "Workspace Automation Pack",
				Price:       5,
				Category:    "automation",
				Description: "A starter package for automating repetitive engineering tasks.",
			},
			{
				Name:        "Incident Triage Guide",
				Price:       6,
				Category:    "operations",
				Description: "A practical guide for diagnosing and resolving production incidents.",
			},
			{
				Name:        "Performance Review Bundle",
				Price:       7,
				Category:    "analysis",
				Description: "A toolkit for profiling, benchmarking, and optimization planning.",
			},
		},
	}
}

func (c *Catalog) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return []mcp.Tool{
		{
			Name:        "listItems",
			Description: "List all item names in the catalog",
			InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}},
		},
		{
			Name:        "getItemDetails",
			Description: "Get detailed information for a catalog item",
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

func (c *Catalog) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	switch params.Name {
	case "listItems":
		return c.listItems(ctx), nil
	case "getItemDetails":
		return c.getItemDetails(ctx, params.Arguments)
	default:
		return mcp.ToolResponse{}, fmt.Errorf("tool %s not found", params.Name)
	}
}

func (c *Catalog) listItems(ctx context.Context) mcp.ToolResponse {
	select {
	case <-ctx.Done():
		return mcp.ToolResponse{
			Content: []mcp.ContentItem{{Type: "text", Text: `{"error":"request cancelled"}`}},
		}
	default:
	}

	names := make([]string, 0, len(c.items))
	for _, item := range c.items {
		names = append(names, item.Name)
	}

	namesJSON, err := json.Marshal(map[string][]string{"names": names})
	if err != nil {
		return mcp.ToolResponse{
			Content: []mcp.ContentItem{{Type: "text", Text: fmt.Sprintf(`{"error":"failed to marshal item names: %s"}`, err.Error())}},
		}
	}

	return mcp.ToolResponse{Content: []mcp.ContentItem{{Type: "text", Text: string(namesJSON)}}}
}

func (c *Catalog) getItemDetails(ctx context.Context, args map[string]any) (mcp.ToolResponse, error) {
	select {
	case <-ctx.Done():
		return mcp.ToolResponse{}, ctx.Err()
	default:
	}

	name, ok := args["name"].(string)
	if !ok {
		return mcp.ToolResponse{}, fmt.Errorf("invalid name parameter: expected string")
	}

	for _, item := range c.items {
		if item.Name == name {
			itemJSON, err := json.Marshal(item)
			if err != nil {
				return mcp.ToolResponse{}, fmt.Errorf("failed to marshal item details: %w", err)
			}
			return mcp.ToolResponse{Content: []mcp.ContentItem{{Type: "text", Text: string(itemJSON)}}}, nil
		}
	}

	return mcp.ToolResponse{}, fmt.Errorf("item not found: %s", name)
}

func (c *Catalog) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	return []mcp.Resource{{URI: "catalog://items", Name: "catalog"}}, nil
}

func (c *Catalog) ReadResource(ctx context.Context, params mcp.ResourceParams) (mcp.ResourceResponse, error) {
	if params.URI == "catalog://items" {
		return c.getCatalogResource()
	}
	return mcp.ResourceResponse{}, fmt.Errorf("resource not found: %s", params.URI)
}

func (c *Catalog) getCatalogResource() (mcp.ResourceResponse, error) {
	itemsJSON, err := json.Marshal(c.items)
	if err != nil {
		return mcp.ResourceResponse{}, fmt.Errorf("failed to marshal catalog: %w", err)
	}
	return mcp.ResourceResponse{Contents: []mcp.ResourceContent{{URI: "catalog://items", Text: string(itemsJSON)}}}, nil
}

func (c *Catalog) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	return []mcp.Prompt{
		{
			Name:        "planRecommendation",
			Description: "Get a recommendation for which catalog item to use",
			Arguments: []mcp.PromptArgument{
				{Name: "budget", Description: "Budget available in dollars", Required: false},
				{Name: "goal", Description: "Primary goal, for example reliability or speed", Required: false},
			},
		},
		{
			Name:        "itemBrief",
			Description: "Get a concise brief for a specific catalog item",
			Arguments:   []mcp.PromptArgument{{Name: "item_name", Description: "Name of the item to summarize", Required: true}},
		},
	}, nil
}

func (c *Catalog) GetPrompt(ctx context.Context, params mcp.PromptParams) (mcp.PromptResponse, error) {
	switch params.Name {
	case "planRecommendation":
		return c.createPlanRecommendationPrompt(params.Arguments), nil
	case "itemBrief":
		return c.createItemBriefPrompt(params.Arguments), nil
	default:
		return mcp.PromptResponse{}, fmt.Errorf("prompt %s not found", params.Name)
	}
}

func (c *Catalog) createPlanRecommendationPrompt(args map[string]any) mcp.PromptResponse {
	budget, hasBudget := args["budget"]
	goal, hasGoal := args["goal"]

	budgetText := ""
	if hasBudget {
		budgetText = fmt.Sprintf(" with a budget of $%v", budget)
	}

	goalText := ""
	if hasGoal {
		goalText = fmt.Sprintf(" focused on %v", goal)
	}

	promptText := fmt.Sprintf(`You are a systems advisor. Recommend the best option for a team%s%s.

Available catalog items:
- Workspace Automation Pack ($5, automation)
- Incident Triage Guide ($6, operations)
- Performance Review Bundle ($7, analysis)

Explain your recommendation and include a short tradeoff analysis.`, budgetText, goalText)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{{
			Role:    "user",
			Content: mcp.MessageContent{Type: "text", Text: promptText},
		}},
	}
}

func (c *Catalog) createItemBriefPrompt(args map[string]any) mcp.PromptResponse {
	itemName, ok := args["item_name"].(string)
	if !ok {
		itemName = "catalog item"
	}

	promptText := fmt.Sprintf(`Provide a concise brief for %s, including:
1. What it is
2. Best use cases
3. Risks or limitations
4. Quick start steps`, itemName)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{{
			Role:    "user",
			Content: mcp.MessageContent{Type: "text", Text: promptText},
		}},
	}
}
