package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
	"github.com/BearHuddleston/mcp-server-example/pkg/spec"
)

type Item struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type Catalog struct {
	items                []Item
	listTool             mcp.Tool
	detailTool           mcp.Tool
	resource             mcp.Resource
	recommendationPrompt mcp.Prompt
	briefPrompt          mcp.Prompt
	recommendationText   string
	briefText            string
}

func NewCatalog() *Catalog {
	return newCatalog(
		[]Item{
			{Name: "Workspace Automation Pack", Price: 5, Category: "automation", Description: "A starter package for automating repetitive engineering tasks."},
			{Name: "Incident Triage Guide", Price: 6, Category: "operations", Description: "A practical guide for diagnosing and resolving production incidents."},
			{Name: "Performance Review Bundle", Price: 7, Category: "analysis", Description: "A toolkit for profiling, benchmarking, and optimization planning."},
		},
		mcp.Tool{Name: "listItems", Description: "List all item names in the catalog", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}, Required: []string{}}},
		mcp.Tool{Name: "getItemDetails", Description: "Get detailed information for a catalog item", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{"name": map[string]string{"type": "string"}}, Required: []string{"name"}}},
		mcp.Resource{URI: "catalog://items", Name: "catalog"},
		mcp.Prompt{Name: "planRecommendation", Description: "Get a recommendation for which catalog item to use", Arguments: []mcp.PromptArgument{{Name: "budget", Description: "Budget available in dollars", Required: false}, {Name: "goal", Description: "Primary goal, for example reliability or speed", Required: false}}},
		mcp.Prompt{Name: "itemBrief", Description: "Get a concise brief for a specific catalog item", Arguments: []mcp.PromptArgument{{Name: "item_name", Description: "Name of the item to summarize", Required: true}}},
		`You are a systems advisor. Recommend the best option for a team%s%s.

Available catalog items:
- Workspace Automation Pack ($5, automation)
- Incident Triage Guide ($6, operations)
- Performance Review Bundle ($7, analysis)

Explain your recommendation and include a short tradeoff analysis.`,
		`Provide a concise brief for %s, including:
1. What it is
2. Best use cases
3. Risks or limitations
4. Quick start steps`,
	)
	}

func NewCatalogFromSpec(sp *spec.Spec) (*Catalog, error) {
	if sp == nil {
		return nil, fmt.Errorf("spec cannot be nil")
	}

	listTool, err := toolByMode(sp.Tools, "list_items")
	if err != nil {
		return nil, err
	}
	detailTool, err := toolByMode(sp.Tools, "get_item_details")
	if err != nil {
		return nil, err
	}

	resource, err := resourceByMode(sp.Resources, "catalog_items")
	if err != nil {
		return nil, err
	}

	recommendationPrompt, err := promptByMode(sp.Prompts, "plan_recommendation")
	if err != nil {
		return nil, err
	}
	briefPrompt, err := promptByMode(sp.Prompts, "item_brief")
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(sp.Items))
	for _, item := range sp.Items {
		items = append(items, Item{
			Name:        item.Name,
			Price:       item.Price,
			Category:    item.Category,
			Description: item.Description,
		})
	}

	return newCatalog(
		items,
		mcp.Tool{Name: listTool.Name, Description: listTool.Description, InputSchema: listTool.InputSchema},
		mcp.Tool{Name: detailTool.Name, Description: detailTool.Description, InputSchema: detailTool.InputSchema},
		mcp.Resource{URI: resource.URI, Name: resource.Name},
		mcp.Prompt{Name: recommendationPrompt.Name, Description: recommendationPrompt.Description, Arguments: recommendationPrompt.Arguments},
		mcp.Prompt{Name: briefPrompt.Name, Description: briefPrompt.Description, Arguments: briefPrompt.Arguments},
		recommendationPrompt.Template,
		briefPrompt.Template,
	), nil
}

func newCatalog(items []Item, listTool mcp.Tool, detailTool mcp.Tool, resource mcp.Resource, recommendationPrompt mcp.Prompt, briefPrompt mcp.Prompt, recommendationText string, briefText string) *Catalog {
	return &Catalog{
		items:                items,
		listTool:             listTool,
		detailTool:           detailTool,
		resource:             resource,
		recommendationPrompt: recommendationPrompt,
		briefPrompt:          briefPrompt,
		recommendationText:   recommendationText,
		briefText:            briefText,
	}
}

func toolByMode(tools []spec.ToolSpec, mode string) (*spec.ToolSpec, error) {
	for i := range tools {
		if tools[i].Mode == mode {
			return &tools[i], nil
		}
	}
	return nil, fmt.Errorf("missing tool mode %q", mode)
}

func resourceByMode(resources []spec.ResourceSpec, mode string) (*spec.ResourceSpec, error) {
	for i := range resources {
		if resources[i].Mode == mode {
			return &resources[i], nil
		}
	}
	return nil, fmt.Errorf("missing resource mode %q", mode)
}

func promptByMode(prompts []spec.PromptSpec, mode string) (*spec.PromptSpec, error) {
	for i := range prompts {
		if prompts[i].Mode == mode {
			return &prompts[i], nil
		}
	}
	return nil, fmt.Errorf("missing prompt mode %q", mode)
}

func (c *Catalog) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return []mcp.Tool{c.listTool, c.detailTool}, nil
}

func (c *Catalog) CallTool(ctx context.Context, params mcp.ToolCallParams) (mcp.ToolResponse, error) {
	switch params.Name {
	case c.listTool.Name:
		return c.listItems(ctx), nil
	case c.detailTool.Name:
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

	argName := "name"
	if len(c.detailTool.InputSchema.Required) > 0 {
		argName = c.detailTool.InputSchema.Required[0]
	}

	name, ok := args[argName].(string)
	if !ok {
		return mcp.ToolResponse{}, fmt.Errorf("invalid %s parameter: expected string", argName)
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
	return []mcp.Resource{c.resource}, nil
}

func (c *Catalog) ReadResource(ctx context.Context, params mcp.ResourceParams) (mcp.ResourceResponse, error) {
	if params.URI == c.resource.URI {
		return c.getCatalogResource()
	}
	return mcp.ResourceResponse{}, fmt.Errorf("resource not found: %s", params.URI)
}

func (c *Catalog) getCatalogResource() (mcp.ResourceResponse, error) {
	itemsJSON, err := json.Marshal(c.items)
	if err != nil {
		return mcp.ResourceResponse{}, fmt.Errorf("failed to marshal catalog: %w", err)
	}
	return mcp.ResourceResponse{Contents: []mcp.ResourceContent{{URI: c.resource.URI, Text: string(itemsJSON)}}}, nil
}

func (c *Catalog) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	return []mcp.Prompt{c.recommendationPrompt, c.briefPrompt}, nil
}

func (c *Catalog) GetPrompt(ctx context.Context, params mcp.PromptParams) (mcp.PromptResponse, error) {
	switch params.Name {
	case c.recommendationPrompt.Name:
		return c.createPlanRecommendationPrompt(params.Arguments), nil
	case c.briefPrompt.Name:
		return c.createItemBriefPrompt(params.Arguments), nil
	default:
		return mcp.PromptResponse{}, fmt.Errorf("prompt %s not found", params.Name)
	}
}

func (c *Catalog) createPlanRecommendationPrompt(args map[string]any) mcp.PromptResponse {
	budgetKey := "budget"
	goalKey := "goal"
	for _, arg := range c.recommendationPrompt.Arguments {
		switch arg.Name {
		case "budget":
			budgetKey = arg.Name
		case "goal", "preference":
			goalKey = arg.Name
		}
	}

	budget, hasBudget := args[budgetKey]
	goal, hasGoal := args[goalKey]

	budgetText := ""
	if hasBudget {
		budgetText = fmt.Sprintf(" with a budget of $%v", budget)
	}

	goalText := ""
	if hasGoal {
		goalText = fmt.Sprintf(" focused on %v", goal)
	}

	promptText := fmt.Sprintf(c.recommendationText, budgetText, goalText)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{{
			Role:    "user",
			Content: mcp.MessageContent{Type: "text", Text: promptText},
		}},
	}
}

func (c *Catalog) createItemBriefPrompt(args map[string]any) mcp.PromptResponse {
	argName := "item_name"
	if len(c.briefPrompt.Arguments) > 0 {
		argName = c.briefPrompt.Arguments[0].Name
	}

	itemName, ok := args[argName].(string)
	if !ok {
		itemName = "catalog item"
	}

	promptText := fmt.Sprintf(c.briefText, itemName)

	return mcp.PromptResponse{
		Messages: []mcp.PromptMessage{{
			Role:    "user",
			Content: mcp.MessageContent{Type: "text", Text: promptText},
		}},
	}
}
