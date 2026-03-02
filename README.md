# MCP Server Template

This repository is a Go MCP server designed to be driven by an AI agent.

It supports a spec-driven onboarding flow so an agent can ask a human what server they want, write a JSON spec, and run this codebase as a reusable MCP template.

This repo is intentionally contract-first: transports and MCP dispatch stay stable while domain behavior is configured through spec files.

## Agent Notice (Read First)

If you are an AI agent, read `AGENTS.md` before doing any work.
Detailed operator instructions are in `skills/mcp-template-operator/SKILL.md`.

## Human Guide

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

## Server Capabilities

### Tools (Default)
- `listItems`: List lookup values with response shape `{"field":"<lookupField>","values":[...]}`.
- `getItemDetails`: Get one item by lookup field (`name` in default config).

### Resources (Default)
- `catalog://items`: Full catalog dataset.

### Prompts (Default)
- `planRecommendation`: Recommendation prompt for selecting an item by budget/goal.
- `itemBrief`: Prompt for generating a concise brief for a specific item.

When `-spec` is provided, tool/resource/prompt names, argument names, and prompt templates come from the spec file.

When `-spec` is provided, the detail lookup field is also driven by `tools[get_item_details].inputSchema.required[0]`.

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
- `items` dynamic objects (free-form fields)
- `tools` with required modes:
  - `list_items`
  - `get_item_details`
- `resources` with required mode:
  - `catalog_items`
- `prompts` with required modes:
  - `plan_recommendation`
  - `item_brief`

Unknown JSON fields are rejected at load time.

Additional validation rules for dynamic item mode:
- `get_item_details.inputSchema.required` must contain exactly one field.
- That required field must exist in `get_item_details.inputSchema.properties`.
- That required field schema must be `{"type":"string"}`.
- Every item must include that lookup field as a non-empty string.
- Lookup values must be unique across items.

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

## Demo Validation

Local smoke test for all demo specs:

```bash
go build -o mcp-template-server ./cmd/mcpserver
for spec in demos/*/mcp-spec.json; do
  timeout 4s ./mcp-template-server -spec "$spec" < /dev/null
done
```

Docker smoke test for all demo specs:

```bash
docker build -t mcp-template-server:local .
base=39300
i=0
for spec in demos/*/mcp-spec.json; do
  name=$(basename "$(dirname "$spec")")
  port=$((base+i))
  cname="demo-${name//[^a-zA-Z0-9]/-}-docker"
  docker rm -f "$cname" >/dev/null 2>&1 || true
  docker run -d --name "$cname" -p "$port:8080" \
    -v "$(pwd)/$spec:/root/spec.json:ro" \
    mcp-template-server:local \
    ./mcp-template-server --transport http --port 8080 --spec /root/spec.json
  sleep 1
  curl -fsS "http://127.0.0.1:$port/health"
  docker rm -f "$cname" >/dev/null
  i=$((i+1))
done
```

## HTTP Endpoints

- `POST /mcp`
- `GET /mcp`
- `DELETE /mcp`
- `GET /health`

Protocol version: `2025-11-25`
