package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

type Spec struct {
	SchemaVersion string         `json:"schemaVersion"`
	Server        ServerSpec     `json:"server"`
	Runtime       RuntimeSpec    `json:"runtime"`
	Items         []ItemSpec     `json:"items"`
	Tools         []ToolSpec     `json:"tools"`
	Resources     []ResourceSpec `json:"resources"`
	Prompts       []PromptSpec   `json:"prompts"`
}

type ServerSpec struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RuntimeSpec struct {
	TransportType  string   `json:"transportType"`
	HTTPPort       int      `json:"httpPort"`
	RequestTimeout string   `json:"requestTimeout"`
	AllowedOrigins []string `json:"allowedOrigins"`
}

type ItemSpec map[string]any

type ToolSpec struct {
	Mode        string          `json:"mode"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema mcp.InputSchema `json:"inputSchema"`
}

type ResourceSpec struct {
	Mode string `json:"mode"`
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type PromptSpec struct {
	Mode        string               `json:"mode"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Arguments   []mcp.PromptArgument `json:"arguments"`
	Template    string               `json:"template"`
}

func LoadFile(path string) (*Spec, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()

	var sp Spec
	if err := decoder.Decode(&sp); err != nil {
		return nil, fmt.Errorf("parse spec file: %w", err)
	}

	if err := sp.Validate(); err != nil {
		return nil, err
	}

	return &sp, nil
}

func (s *Spec) Validate() error {
	if s.SchemaVersion != "v1" {
		return fmt.Errorf("invalid schemaVersion %q, expected \"v1\"", s.SchemaVersion)
	}

	lookupKey, err := validateTools(s.Tools)
	if err != nil {
		return err
	}
	if err := validateItems(s.Items, lookupKey); err != nil {
		return err
	}
	if err := validateResources(s.Resources); err != nil {
		return err
	}
	if err := validatePrompts(s.Prompts); err != nil {
		return err
	}
	if err := validateRuntime(s.Runtime); err != nil {
		return err
	}

	return nil
}

func validateTools(tools []ToolSpec) (string, error) {
	if len(tools) == 0 {
		return "", fmt.Errorf("spec must include tool definitions")
	}

	validModes := []string{"list_items", "get_item_details"}
	modeSeen := make(map[string]struct{}, len(validModes))
	nameSeen := make(map[string]struct{}, len(tools))
	lookupKey := ""

	for _, tool := range tools {
		if !slices.Contains(validModes, tool.Mode) {
			return "", fmt.Errorf("invalid tool mode %q", tool.Mode)
		}
		if _, ok := modeSeen[tool.Mode]; ok {
			return "", fmt.Errorf("duplicate tool mode %q", tool.Mode)
		}
		modeSeen[tool.Mode] = struct{}{}

		if !toolNamePattern.MatchString(tool.Name) {
			return "", fmt.Errorf("invalid tool name %q", tool.Name)
		}
		if _, ok := nameSeen[tool.Name]; ok {
			return "", fmt.Errorf("duplicate tool name %q", tool.Name)
		}
		nameSeen[tool.Name] = struct{}{}

		if strings.TrimSpace(tool.Description) == "" {
			return "", fmt.Errorf("tool %q description cannot be empty", tool.Name)
		}
		if strings.TrimSpace(tool.InputSchema.Type) == "" {
			return "", fmt.Errorf("tool %q inputSchema.type cannot be empty", tool.Name)
		}
		if tool.Mode == "get_item_details" {
			if len(tool.InputSchema.Required) != 1 || strings.TrimSpace(tool.InputSchema.Required[0]) == "" {
				return "", fmt.Errorf("tool %q must define exactly one required lookup field", tool.Name)
			}
			lookupKey = strings.TrimSpace(tool.InputSchema.Required[0])

			prop, ok := tool.InputSchema.Properties[lookupKey]
			if !ok {
				return "", fmt.Errorf("tool %q required lookup field %q must exist in inputSchema.properties", tool.Name, lookupKey)
			}

			propType, ok := schemaType(prop)
			if !ok || strings.TrimSpace(propType) != "string" {
				return "", fmt.Errorf("tool %q lookup field %q schema type must be string", tool.Name, lookupKey)
			}
		}
	}

	for _, mode := range validModes {
		if _, ok := modeSeen[mode]; !ok {
			return "", fmt.Errorf("missing required tool mode %q", mode)
		}
	}
	if lookupKey == "" {
		return "", fmt.Errorf("missing lookup field in get_item_details tool definition")
	}

	return lookupKey, nil
}

func validateItems(items []ItemSpec, lookupKey string) error {
	if len(items) == 0 {
		return fmt.Errorf("spec must include at least one item")
	}

	seen := make(map[string]struct{}, len(items))
	for i, item := range items {
		if len(item) == 0 {
			return fmt.Errorf("item at index %d cannot be empty", i)
		}

		lookupValue, ok := item[lookupKey]
		if !ok {
			return fmt.Errorf("item at index %d missing lookup field %q", i, lookupKey)
		}

		lookupString, ok := lookupValue.(string)
		if !ok || strings.TrimSpace(lookupString) == "" {
			return fmt.Errorf("item at index %d field %q must be a non-empty string", i, lookupKey)
		}

		if _, exists := seen[lookupString]; exists {
			return fmt.Errorf("duplicate item lookup value %q for field %q", lookupString, lookupKey)
		}
		seen[lookupString] = struct{}{}
	}

	return nil
}

func schemaType(schema any) (string, bool) {
	switch typed := schema.(type) {
	case map[string]any:
		value, ok := typed["type"].(string)
		if !ok {
			return "", false
		}
		return value, true
	case map[string]string:
		value, ok := typed["type"]
		if !ok {
			return "", false
		}
		return value, true
	default:
		return "", false
	}
}

func validateResources(resources []ResourceSpec) error {
	if len(resources) == 0 {
		return fmt.Errorf("spec must include resource definitions")
	}

	validModes := []string{"catalog_items"}
	modeSeen := make(map[string]struct{}, len(validModes))
	uriSeen := make(map[string]struct{}, len(resources))

	for _, resource := range resources {
		if !slices.Contains(validModes, resource.Mode) {
			return fmt.Errorf("invalid resource mode %q", resource.Mode)
		}
		if _, ok := modeSeen[resource.Mode]; ok {
			return fmt.Errorf("duplicate resource mode %q", resource.Mode)
		}
		modeSeen[resource.Mode] = struct{}{}

		if strings.TrimSpace(resource.URI) == "" {
			return fmt.Errorf("resource uri cannot be empty")
		}
		if _, ok := uriSeen[resource.URI]; ok {
			return fmt.Errorf("duplicate resource uri %q", resource.URI)
		}
		uriSeen[resource.URI] = struct{}{}

		if strings.TrimSpace(resource.Name) == "" {
			return fmt.Errorf("resource name cannot be empty")
		}
	}

	for _, mode := range validModes {
		if _, ok := modeSeen[mode]; !ok {
			return fmt.Errorf("missing required resource mode %q", mode)
		}
	}

	return nil
}

func validatePrompts(prompts []PromptSpec) error {
	if len(prompts) == 0 {
		return fmt.Errorf("spec must include prompt definitions")
	}

	validModes := []string{"plan_recommendation", "item_brief"}
	modeSeen := make(map[string]struct{}, len(validModes))
	nameSeen := make(map[string]struct{}, len(prompts))

	for _, prompt := range prompts {
		if !slices.Contains(validModes, prompt.Mode) {
			return fmt.Errorf("invalid prompt mode %q", prompt.Mode)
		}
		if _, ok := modeSeen[prompt.Mode]; ok {
			return fmt.Errorf("duplicate prompt mode %q", prompt.Mode)
		}
		modeSeen[prompt.Mode] = struct{}{}

		if !toolNamePattern.MatchString(prompt.Name) {
			return fmt.Errorf("invalid prompt name %q", prompt.Name)
		}
		if _, ok := nameSeen[prompt.Name]; ok {
			return fmt.Errorf("duplicate prompt name %q", prompt.Name)
		}
		nameSeen[prompt.Name] = struct{}{}

		if strings.TrimSpace(prompt.Description) == "" {
			return fmt.Errorf("prompt %q description cannot be empty", prompt.Name)
		}
		if strings.TrimSpace(prompt.Template) == "" {
			return fmt.Errorf("prompt %q template cannot be empty", prompt.Name)
		}
	}

	for _, mode := range validModes {
		if _, ok := modeSeen[mode]; !ok {
			return fmt.Errorf("missing required prompt mode %q", mode)
		}
	}

	return nil
}

func validateRuntime(runtime RuntimeSpec) error {
	if strings.TrimSpace(runtime.TransportType) != "" {
		transportType := strings.ToLower(strings.TrimSpace(runtime.TransportType))
		if transportType != "stdio" && transportType != "http" {
			return fmt.Errorf("invalid runtime transportType %q", runtime.TransportType)
		}
	}
	if runtime.HTTPPort != 0 && (runtime.HTTPPort < 1 || runtime.HTTPPort > 65535) {
		return fmt.Errorf("invalid runtime httpPort %d", runtime.HTTPPort)
	}
	if strings.TrimSpace(runtime.RequestTimeout) != "" {
		duration, err := time.ParseDuration(runtime.RequestTimeout)
		if err != nil {
			return fmt.Errorf("invalid runtime requestTimeout: %w", err)
		}
		if duration <= 0 {
			return fmt.Errorf("runtime requestTimeout must be positive")
		}
	}

	return nil
}
