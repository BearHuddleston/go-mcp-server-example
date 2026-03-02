# Demo Specs

This directory contains ready-to-run demo specs showing common ways to use this MCP template.

## Demos

- `service-catalog/mcp-spec.json`: Internal service catalog assistant
- `playbook-recommender/mcp-spec.json`: Recommendation and planning assistant
- `knowledge-front-door/mcp-spec.json`: Team knowledge access via curated resources
- `incident-runbook-template/mcp-spec.json`: Incident and runbook guidance template
- `edge-secure/mcp-spec.json`: Security-focused edge/local deployment profile

## Run Any Demo

```bash
go build -o mcp-template-server ./cmd/mcpserver
./mcp-template-server -spec ./demos/service-catalog/mcp-spec.json
```

For HTTP transport:

```bash
./mcp-template-server -transport http -port 8080 -spec ./demos/service-catalog/mcp-spec.json
```
