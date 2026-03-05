# Contributing

Thanks for your interest in improving this project.

## Before You Start

- Read `README.md` for runtime and spec behavior.
- Read `AGENTS.md` and `skills/mcp-template-operator/SKILL.md` for agent/operator workflows.

## Development Workflow

1. Fork and create a topic branch.
2. Make focused, atomic changes.
3. Run checks locally:

```bash
go test ./...
go build ./cmd/mcpserver
```

4. If changing behavior, update docs and demos as needed.
5. Open a PR with:
   - Problem statement
   - What changed and why
   - Verification evidence (tests/build/output)

## Style Expectations

- Keep transport/core contracts stable unless change is intentional and documented.
- In spec mode, treat `mcp-spec*.json` as contract source of truth.
- Prefer capability discovery over hardcoded capability names.

## Reporting Issues

- Use GitHub Issues for bugs and feature requests.
- Use `SECURITY.md` process for vulnerabilities.
