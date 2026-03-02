# Demo Specs

This directory contains ready-to-run demo specs showing common ways to use this MCP template.

## Demos

- `service-catalog/mcp-spec.json`: Internal service catalog assistant
- `playbook-recommender/mcp-spec.json`: Recommendation and planning assistant
- `knowledge-front-door/mcp-spec.json`: Team knowledge access via curated resources
- `incident-runbook-template/mcp-spec.json`: Incident and runbook guidance template
- `edge-secure/mcp-spec.json`: Security-focused edge/local deployment profile

## Lookup Fields Per Demo

Each demo uses a different dynamic lookup key for `get_item_details`:

- `service-catalog`: `service_name`
- `playbook-recommender`: `playbook_name`
- `knowledge-front-door`: `source_name`
- `incident-runbook-template`: `incident_name`
- `edge-secure`: `profile_name`

The key above is the required field in `tools[mode=get_item_details].inputSchema.required[0]` and must exist in every item.

## Run Any Demo

```bash
go build -o mcp-template-server ./cmd/mcpserver
./mcp-template-server -spec ./demos/service-catalog/mcp-spec.json
```

For HTTP transport:

```bash
./mcp-template-server -transport http -port 8080 -spec ./demos/service-catalog/mcp-spec.json
```

Run all demos quickly:

```bash
for spec in demos/*/mcp-spec.json; do
  timeout 4s ./mcp-template-server -spec "$spec" < /dev/null
done
```
