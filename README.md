# MCP Server Template

This repository is a Go MCP server designed to be driven by an AI agent.

It supports a spec-driven onboarding flow so an agent can ask a human what server they want, write a JSON spec, and run this codebase as a reusable MCP template.

## Common Use Cases

- Internal service catalog assistant for onboarding and discovery.
- Playbook recommender that drafts plans from budget/goal inputs.
- Team knowledge front door exposing curated MCP resources.
- Template for domain-specific MCP servers (HR, platform docs, incident runbooks).
- Secure local or edge deployment with hardened Docker runtime defaults.

## Demo Specs

You can try prebuilt demos for these use cases under `demos/`:

- `demos/service-catalog/mcp-spec.json`
- `demos/playbook-recommender/mcp-spec.json`
- `demos/knowledge-front-door/mcp-spec.json`
- `demos/incident-runbook-template/mcp-spec.json`
- `demos/edge-secure/mcp-spec.json`

See `demos/README.md` for run commands.

## Quick Start

```bash
go build -o mcp-template-server ./cmd/mcpserver

# Run defaults from in-code catalog
./mcp-template-server

# Run from spec template
./mcp-template-server -spec ./mcp-spec.example.json
```

## Security Recommendation

For production or shared environments, prefer running this server in Docker instead of directly on the host.

Why:
- Container isolation reduces host-level blast radius.
- You can enforce runtime restrictions (`--read-only`, dropped capabilities, no-new-privileges).
- Deployment is more reproducible across environments.

Hardened container example:

```bash
docker run --rm -p 8080:8080 \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges:true \
  -v "$(pwd)/mcp-spec.example.json:/root/mcp-spec.example.json:ro" \
  mcp-template-server:local \
  ./mcp-template-server --transport http --port 8080 --spec /root/mcp-spec.example.json
```

## Onboarding Workflow

1. Agent asks the human what MCP server they want to build (tools, resources, prompts, and item data).
2. Agent writes a spec JSON file using schema version `v1`.
3. Server starts with `-spec <path>` and loads that spec at boot.
4. `pkg/spec` validates required modes and shape, then `pkg/handlers` builds a configured catalog handler.
5. MCP core and transports stay unchanged; only behavior/content is configured by spec.

Use `mcp-spec.example.json` in the repo root as a starting template.

## AI Agent Prompt

Use this prompt when connecting an AI agent to this server.

Agent-agnostic skill index is in `AGENTS.md`; the operator skill is `skills/mcp-template-operator/SKILL.md`.

```text
You are an AI assistant connected to an MCP server named "MCP Template Server".

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
2. Discover available capabilities via tools/list, resources/list, and prompts/list.
3. Choose tool/resource/prompt names from discovery results (do not hardcode names).
4. Call the list-style tool first to enumerate options.
5. Call the details-style tool for selected items.
6. Use matching resource and prompt endpoints for broader context and user-facing output.

Output style:
- Use short sections: "What I checked", "Result", "Next step".
- Include JSON snippets only when useful.
- Avoid filler text.
```

## Server Capabilities

### Tools (Default)
- `listItems`: List available catalog item names.
- `getItemDetails`: Get details for one catalog item (`name` required).

### Resources (Default)
- `catalog://items`: Full catalog dataset.

### Prompts (Default)
- `planRecommendation`: Recommendation prompt for selecting an item by budget/goal.
- `itemBrief`: Prompt for generating a concise brief for a specific item.

When `-spec` is provided, tool/resource/prompt names, argument names, and prompt templates come from the spec file.

## Run Locally

```bash
go build -o mcp-template-server ./cmd/mcpserver

# stdio (default)
./mcp-template-server

# HTTP transport
./mcp-template-server -transport http -port 8080

# Spec-driven behavior (stdio)
./mcp-template-server -spec ./mcp-spec.example.json

# Spec-driven behavior (HTTP)
./mcp-template-server -transport http -port 8080 -spec ./mcp-spec.example.json
```

If the spec is invalid, startup fails with a validation error.

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

## Development

```bash
go test ./...
go build ./cmd/mcpserver
```

## Docker

```bash
# Build image
docker build -t mcp-template-server:local .

# Run default container command (HTTP on port 8080)
docker run --rm -p 8080:8080 mcp-template-server:local

# Run with spec-driven behavior
docker run --rm -p 8080:8080 \
  -v "$(pwd)/mcp-spec.example.json:/root/mcp-spec.example.json:ro" \
  mcp-template-server:local \
  ./mcp-template-server --transport http --port 8080 --spec /root/mcp-spec.example.json
```

## HTTP Endpoints

- `POST /mcp`
- `GET /mcp`
- `DELETE /mcp`
- `GET /health`

Protocol version: `2025-11-25`
