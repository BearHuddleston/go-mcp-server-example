# MCP Server Example

This repository is a Go MCP server designed to be driven by an AI agent.

It now supports a spec-driven onboarding flow so an agent can ask a human what server they want, write a JSON spec, and run this codebase as a template.

## Onboarding Workflow

1. Agent asks the human what MCP server they want to build (tools, resources, prompts, and item data).
2. Agent writes a spec JSON file using schema version `v1`.
3. Server starts with `-spec <path>` and loads that spec at boot.
4. `pkg/spec` validates required modes and shape, then `pkg/handlers` builds a configured catalog handler.
5. MCP core and transports stay unchanged; only behavior/content is configured by spec.

Use `mcp-spec.example.json` in the repo root as a starting template.

## AI Agent Prompt

Use this prompt when connecting an AI agent to this server.

```text
You are an AI assistant connected to an MCP server named "MCP Example Server".

Goal:
- Help the user complete software engineering tasks using MCP tools, resources, and prompts.

Behavior rules:
1. Start by discovering capabilities with tools/list, resources/list, and prompts/list.
2. Before calling a tool, explain in one sentence what data you are about to fetch.
3. Prefer concrete tool calls over assumptions.
4. For any recommendation, include at least one direct data point from a tool/resource result.
5. If a requested item is missing, report it clearly and suggest the closest available option.
6. Keep responses concise and actionable.

Workflow:
1. Initialize MCP session.
2. List tools and resources.
3. Use listItems to discover catalog options.
4. Use getItemDetails(name=...) for specific item facts.
5. Use resource catalog://items when broad context is needed.
6. Use prompts planRecommendation or itemBrief when drafting user-facing guidance.

Output style:
- Use short sections: "What I checked", "Result", "Next step".
- Include JSON snippets only when useful.
- Avoid filler text.
```

## Server Capabilities

### Tools
- `listItems`: List available catalog item names.
- `getItemDetails`: Get details for one catalog item (`name` required).

### Resources
- `catalog://items`: Full catalog dataset.

### Prompts
- `planRecommendation`: Recommendation prompt for selecting an item by budget/goal.
- `itemBrief`: Prompt for generating a concise brief for a specific item.

When `-spec` is provided, tool/resource/prompt names and prompt templates come from the spec file.

## Run Locally

```bash
go build -o mcpserver ./cmd/mcpserver

# stdio (default)
./mcpserver

# HTTP transport
./mcpserver -transport http -port 8080

# Spec-driven behavior (stdio)
./mcpserver -spec ./mcp-spec.example.json

# Spec-driven behavior (HTTP)
./mcpserver -transport http -port 8080 -spec ./mcp-spec.example.json
```

## Spec Schema

`mcp-spec.json` must include:

- `schemaVersion` (currently `"v1"`)
- `server` metadata
- `runtime` defaults (`transportType`, optional `httpPort`, optional `requestTimeout`, optional `allowedOrigins`)
- `items` catalog entries
- `tools` with required modes:
  - `list_items`
  - `get_item_details`
- `resources` with required mode:
  - `catalog_items`
- `prompts` with required modes:
  - `plan_recommendation`
  - `item_brief`

Unknown JSON fields are rejected at load time.

## HTTP Endpoints

- `POST /mcp`
- `GET /mcp`
- `DELETE /mcp`
- `GET /health`

Protocol version: `2025-11-25`
