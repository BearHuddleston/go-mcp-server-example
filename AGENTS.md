# Agent Guidance

This repository is a spec-driven MCP server template. Use the skills index below to apply repeatable operating procedures.

## Skills Index

- `mcp-template-operator`: `skills/mcp-template-operator/SKILL.md`
  - Run and validate the MCP server.
  - Use capability discovery (`tools/list`, `resources/list`, `prompts/list`).
  - Prefer hardened Docker execution for safer runtime defaults.

## General Rules

- Prefer capability-driven usage over hardcoded tool/resource/prompt names.
- In spec mode, treat `mcp-spec*.json` as the contract source of truth.
- Surface concrete verification evidence (startup logs, health checks, MCP list calls).
- Respect dynamic item schema contract:
  - `get_item_details.inputSchema.required` must define exactly one lookup field.
  - The lookup field must exist in item records and be unique/non-empty string values.
