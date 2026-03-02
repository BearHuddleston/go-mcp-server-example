---
name: mcp-template-operator
description: Use when running, validating, or troubleshooting this repo's MCP server, especially with spec-driven capability names and secure Docker runtime requirements.
---

# MCP Template Operator

## Overview

Use this skill to operate this repository's MCP server safely and consistently.
Core principle: discover capabilities at runtime, do not hardcode names in spec-driven mode.

## When to Use

Use when tasks involve:

- Starting the MCP server (local or Docker)
- Validating tools/resources/prompts end-to-end
- Troubleshooting startup, transport, or capability issues
- Running with hardened container defaults

Do not use for unrelated application feature development.

## Quick Reference

- Start server with default catalog or `--spec` file.
- Discover capabilities via `tools/list`, `resources/list`, `prompts/list`.
- Use list-style capability before details-style capability.
- Treat list response as lookup metadata: `{"field":"<lookupField>","values":[...]}`.
- Include at least one concrete data point from tool/resource output in recommendations.

Dynamic schema rule:
- `get_item_details.inputSchema.required[0]` is the lookup field and must match item keys.

## Implementation

```bash
# Local binary path
go build -o mcp-template-server ./cmd/mcpserver
./mcp-template-server --transport http --port 8080 --spec ./mcp-spec.example.json

# Docker path (hardened)
docker build -t mcp-template-server:local .
docker run --rm -p 8080:8080 \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges:true \
  -v "$(pwd)/mcp-spec.example.json:/root/mcp-spec.example.json:ro" \
  mcp-template-server:local \
  ./mcp-template-server --transport http --port 8080 --spec /root/mcp-spec.example.json
```

## Common Mistakes

- Hardcoding tool/resource/prompt names instead of discovery calls
- Running without read-only mounts for spec files in Docker
- Skipping health and endpoint checks when transport starts but calls fail

## Failure Handling

- Startup errors: classify as config/spec/port/auth/runtime.
- Spec validation errors: report exact failing field or required mode.
- Runtime request errors: verify `GET /health` and MCP endpoint accessibility before deeper debugging.
