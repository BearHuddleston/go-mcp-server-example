# Demo Specs

This directory contains ready-to-run demo specs showing common ways to use this MCP template.

## Demos

- `service-catalog/mcp-spec.json`: Internal service catalog assistant
- `playbook-recommender/mcp-spec.json`: Recommendation and planning assistant
- `knowledge-front-door/mcp-spec.json`: Team knowledge access via curated resources
- `incident-runbook-template/mcp-spec.json`: Incident and runbook guidance template
- `edge-secure/mcp-spec.json`: Security-focused edge/local deployment profile

## Use Case Breakdown

### Service Catalog (`service-catalog/mcp-spec.json`)
- **Why:** give teams a single MCP-accessible index of platform services.
- **Where:** internal developer portals, onboarding assistants, platform support chats.
- **When:** new team setup, architecture planning, dependency discovery.
- **What:** service inventory with ownership/tier metadata and service-specific prompts.
- **How:** lookup by `service_name`, then use details + prompts for recommendations.

### Playbook Recommender (`playbook-recommender/mcp-spec.json`)
- **Why:** turn broad goals into actionable improvement plans.
- **Where:** engineering management workflows, planning reviews, transformation programs.
- **When:** quarter planning, reliability initiatives, delivery optimization efforts.
- **What:** curated playbooks plus recommendation/brief prompt templates.
- **How:** lookup by `playbook_name`, then use prompt outputs for plan drafts.

### Knowledge Front Door (`knowledge-front-door/mcp-spec.json`)
- **Why:** reduce documentation sprawl and speed up context retrieval.
- **Where:** internal copilots, support desks, onboarding assistants.
- **When:** policy lookup, standards questions, "where do I find..." moments.
- **What:** curated knowledge sources exposed as tools/resources/prompts.
- **How:** lookup by `source_name`, then route users to the right source quickly.

### Incident Runbook Template (`incident-runbook-template/mcp-spec.json`)
- **Why:** standardize incident guidance and improve response consistency.
- **Where:** SRE operations, incident channels, on-call tooling assistants.
- **When:** active incidents, postmortems, runbook rehearsal/training.
- **What:** incident scenario catalog with triage-oriented prompts.
- **How:** lookup by `incident_name`, then generate response briefs and action plans.

### Edge Secure (`edge-secure/mcp-spec.json`)
- **Why:** enforce security-first rollout guidance for constrained deployments.
- **Where:** edge environments, regulated workloads, shared infrastructure contexts.
- **When:** deployment hardening, compliance checks, secure release preparation.
- **What:** security profile catalog with control-focused recommendation prompts.
- **How:** lookup by `profile_name`, then apply controls and validation steps from outputs.

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
