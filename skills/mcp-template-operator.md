# MCP Template Operator Skill (Agent-Agnostic)

Use this playbook when operating this repository's MCP server with any AI agent platform.

## Intent

- Run and use the MCP template server safely and consistently.
- Prefer secure containerized execution.
- Keep usage capability-driven so spec-based naming changes do not break workflows.

## Operating Pattern

1. Start the server (prefer Docker) with defaults or `--spec /path/to/spec.json`.
2. Initialize MCP session.
3. Discover capabilities dynamically:
   - `tools/list`
   - `resources/list`
   - `prompts/list`
4. Do not hardcode tool/resource/prompt names in spec-driven mode.
5. Use list-style capability first, then details-style capability.
6. For recommendations, include at least one direct data point from tool/resource output.

## Security Defaults

- Prefer Docker over host execution.
- Use reduced-privilege runtime where possible:
  - `--read-only`
  - `--cap-drop=ALL`
  - `--security-opt=no-new-privileges:true`
- Mount spec/config files read-only.

## Canonical Commands

```bash
# Local binary
go build -o mcp-template-server ./cmd/mcpserver
./mcp-template-server --transport http --port 8080 --spec ./mcp-spec.example.json

# Docker (hardened)
docker build -t mcp-template-server:local .
docker run --rm -p 8080:8080 \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges:true \
  -v "$(pwd)/mcp-spec.example.json:/root/mcp-spec.example.json:ro" \
  mcp-template-server:local \
  ./mcp-template-server --transport http --port 8080 --spec /root/mcp-spec.example.json
```

## Failure Handling

- If startup fails, report exact error and classify: config/spec/port/auth/runtime.
- If spec validation fails, report the exact field or required mode that failed.
- If transport starts but requests fail, verify endpoint and health first (`GET /health`, then `/mcp`).
