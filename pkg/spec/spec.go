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

type ItemSpec struct {
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

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

	if len(s.Items) == 0 {
		return fmt.Errorf("spec must include at least one item")
	}

	itemNames := make(map[string]struct{}, len(s.Items))
	for _, item := range s.Items {
		if strings.TrimSpace(item.Name) == "" {
			return fmt.Errorf("item name cannot be empty")
		}
		if _, ok := itemNames[item.Name]; ok {
			return fmt.Errorf("duplicate item name %q", item.Name)
		}
		itemNames[item.Name] = struct{}{}
	}

	if err := validateTools(s.Tools); err != nil {
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

func validateTools(tools []ToolSpec) error {
	if len(tools) == 0 {
		return fmt.Errorf("spec must include tool definitions")
	}

	validModes := []string{"list_items", "get_item_details"}
	modeSeen := make(map[string]struct{}, len(validModes))
	nameSeen := make(map[string]struct{}, len(tools))

	for _, tool := range tools {
		if !slices.Contains(validModes, tool.Mode) {
			return fmt.Errorf("invalid tool mode %q", tool.Mode)
		}
		if _, ok := modeSeen[tool.Mode]; ok {
			return fmt.Errorf("duplicate tool mode %q", tool.Mode)
		}
		modeSeen[tool.Mode] = struct{}{}

		if !toolNamePattern.MatchString(tool.Name) {
			return fmt.Errorf("invalid tool name %q", tool.Name)
		}
		if _, ok := nameSeen[tool.Name]; ok {
			return fmt.Errorf("duplicate tool name %q", tool.Name)
		}
		nameSeen[tool.Name] = struct{}{}

		if strings.TrimSpace(tool.Description) == "" {
			return fmt.Errorf("tool %q description cannot be empty", tool.Name)
		}
		if strings.TrimSpace(tool.InputSchema.Type) == "" {
			return fmt.Errorf("tool %q inputSchema.type cannot be empty", tool.Name)
		}
	}

	for _, mode := range validModes {
		if _, ok := modeSeen[mode]; !ok {
			return fmt.Errorf("missing required tool mode %q", mode)
		}
	}

	return nil
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
