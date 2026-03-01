package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

func TestLoadFile(t *testing.T) {
	t.Run("valid spec", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mcp-spec.json")
		content := `{
		  "schemaVersion": "v1",
		  "server": {"name": "Template MCP", "version": "1.0.0"},
		  "runtime": {"transportType": "http", "httpPort": 8080, "requestTimeout": "30s", "allowedOrigins": ["http://localhost:3000"]},
		  "items": [
		    {"name": "Item A", "price": 5, "category": "starter", "description": "First item"}
		  ],
		  "tools": [
		    {"mode": "list_items", "name": "listItems", "description": "List items", "inputSchema": {"type": "object", "properties": {}, "required": []}},
		    {"mode": "get_item_details", "name": "getItemDetails", "description": "Get item details", "inputSchema": {"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}}
		  ],
		  "resources": [
		    {"mode": "catalog_items", "uri": "catalog://items", "name": "catalog"}
		  ],
		  "prompts": [
		    {"mode": "plan_recommendation", "name": "planRecommendation", "description": "Plan prompt", "arguments": [{"name": "budget", "description": "Budget", "required": false}], "template": "Plan for team%s%s"},
		    {"mode": "item_brief", "name": "itemBrief", "description": "Brief prompt", "arguments": [{"name": "item_name", "description": "Item", "required": true}], "template": "Brief for %s"}
		  ]
		}`

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		sp, err := LoadFile(path)
		if err != nil {
			t.Fatalf("LoadFile failed: %v", err)
		}
		if sp.SchemaVersion != "v1" {
			t.Fatalf("unexpected schema version: %s", sp.SchemaVersion)
		}
		if len(sp.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(sp.Items))
		}
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mcp-spec.json")
		content := `{
		  "schemaVersion": "v1",
		  "server": {"name": "Template MCP", "version": "1.0.0"},
		  "runtime": {},
		  "items": [{"name": "Item A", "price": 5, "category": "starter", "description": "First item"}],
		  "tools": [
		    {"mode": "list_items", "name": "listItems", "description": "List items", "inputSchema": {"type": "object", "properties": {}, "required": []}},
		    {"mode": "get_item_details", "name": "getItemDetails", "description": "Get item details", "inputSchema": {"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}}
		  ],
		  "resources": [{"mode": "catalog_items", "uri": "catalog://items", "name": "catalog"}],
		  "prompts": [
		    {"mode": "plan_recommendation", "name": "planRecommendation", "description": "Plan prompt", "arguments": [{"name": "budget", "description": "Budget", "required": false}], "template": "Plan for team%s%s"},
		    {"mode": "item_brief", "name": "itemBrief", "description": "Brief prompt", "arguments": [{"name": "item_name", "description": "Item", "required": true}], "template": "Brief for %s"}
		  ],
		  "extra": true
		}`

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write spec file: %v", err)
		}

		_, err := LoadFile(path)
		if err == nil {
			t.Fatal("expected unknown field error")
		}
		if !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("expected unknown field error, got %v", err)
		}
	})
}

func TestSpecValidate(t *testing.T) {
	t.Run("invalid schema version", func(t *testing.T) {
		sp := &Spec{SchemaVersion: "v2"}
		err := sp.Validate()
		if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
			t.Fatalf("expected schemaVersion error, got %v", err)
		}
	})

	t.Run("invalid runtime transport", func(t *testing.T) {
		sp := validSpecForValidate()
		sp.Runtime.TransportType = "grpc"
		err := sp.Validate()
		if err == nil || !strings.Contains(err.Error(), "transportType") {
			t.Fatalf("expected transportType error, got %v", err)
		}
	})
}

func validSpecForValidate() *Spec {
	return &Spec{
		SchemaVersion: "v1",
		Items: []ItemSpec{{
			Name:        "Item A",
			Price:       5,
			Category:    "starter",
			Description: "First item",
		}},
		Tools: []ToolSpec{
			{Mode: "list_items", Name: "listItems", Description: "List items", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{}, Required: []string{}}},
			{Mode: "get_item_details", Name: "getItemDetails", Description: "Get details", InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]any{"name": map[string]string{"type": "string"}}, Required: []string{"name"}}},
		},
		Resources: []ResourceSpec{{Mode: "catalog_items", URI: "catalog://items", Name: "catalog"}},
		Prompts: []PromptSpec{
			{Mode: "plan_recommendation", Name: "planRecommendation", Description: "Plan prompt", Template: "Plan for%s%s"},
			{Mode: "item_brief", Name: "itemBrief", Description: "Brief prompt", Template: "Brief for %s"},
		},
	}
}
