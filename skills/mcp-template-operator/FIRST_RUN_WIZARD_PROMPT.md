# First-Run Wizard Prompt (Reusable)

Use this prompt when onboarding a user for the first time or when no spec file exists yet.

## Intent

- Guide users through a short setup wizard.
- Produce a valid `mcp-spec.json` for this repository.
- Keep follow-up interactions in direct-edit mode unless the user explicitly asks to run the wizard again.

## Trigger Conditions

Run wizard mode when any of these are true:

- User says they are new or asks to set up from scratch.
- No spec file exists in the target path.
- Existing spec is missing required `v1` contract sections.

Otherwise, use direct-edit mode (apply requested changes to current spec).

## Wizard Questions (Ask in Order)

1. **Domain & scope**
   - "What domain should this MCP server cover (e.g., service catalog, runbooks, policies)?"
2. **Lookup field**
   - "What key should uniquely identify each item (e.g., `service_name`, `policy_id`)?"
3. **Tool names**
   - "Preferred names for list/details tools?"
   - Defaults: `listItems`, `getItemDetails`
4. **Resource name/URI**
   - "Preferred resource name and URI?"
   - Defaults: `catalog`, `catalog://items`
5. **Prompt names and intent**
   - "Preferred recommendation and brief prompt names?"
   - Defaults: `planRecommendation`, `itemBrief`
6. **Runtime mode**
   - "Run with `stdio` or `http` by default?"
   - If `http`, ask for port and allowed origins.
7. **Initial items**
   - Ask for at least 3 items using the lookup field plus 2-4 additional fields per item.

## Output Contract

After questions are answered, the agent must:

1. Show a concise plan summary.
2. Generate/update `mcp-spec.json` with schemaVersion `v1`.
3. Validate against repository dynamic schema rules:
   - `get_item_details.inputSchema.required` has exactly one key.
   - That key exists in `get_item_details.inputSchema.properties` with type `string`.
   - Every item has that key as a non-empty string.
   - Lookup values are unique.
4. Run a local smoke validation command.
5. Show the resulting capability names and next commands to run.

## Reusable Prompt Text

```text
You are setting up this MCP server for a first-time user.

Mode:
- If no valid spec exists, run setup wizard mode.
- If a valid spec exists, use direct-edit mode unless user asks for wizard.

Wizard behavior:
1) Ask exactly one question at a time using the question order in FIRST_RUN_WIZARD_PROMPT.md.
2) Keep each question short and concrete.
3) Use sensible defaults when the user is unsure and clearly state them.
4) Build or update mcp-spec.json to schemaVersion v1.
5) Enforce dynamic lookup-field contract:
   - exactly one required lookup key for get_item_details
   - lookup key exists in inputSchema.properties with type string
   - every item includes that key with unique non-empty string values
6) After writing spec, run a smoke check and report:
   - server start check result
   - tools/resources/prompts names configured
   - next 1-2 commands user can run.

Direct-edit mode behavior:
- Apply requested changes to existing spec.
- Preserve current naming conventions unless user asks to rename.
- Re-validate and report minimal diff summary.
```
