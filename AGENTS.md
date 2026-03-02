# Agent Guidance

This file defines repository-level policy for coding agents.
It is not the MCP usage onboarding guide.

## Skills Index

- `mcp-template-operator`: `skills/mcp-template-operator/SKILL.md`
  - Use this for MCP runtime onboarding and operational playbook.
  - First-time setup wizard prompt: `skills/mcp-template-operator/FIRST_RUN_WIZARD_PROMPT.md`.

## General Rules

- Prefer capability-driven usage over hardcoded tool/resource/prompt names.
- In spec mode, treat `mcp-spec*.json` as the contract source of truth.
- Surface concrete verification evidence (startup logs, health checks, MCP list calls).
- Respect dynamic item schema contract:
  - `get_item_details.inputSchema.required` must define exactly one lookup field.
  - The lookup field must exist in item records and be unique/non-empty string values.
