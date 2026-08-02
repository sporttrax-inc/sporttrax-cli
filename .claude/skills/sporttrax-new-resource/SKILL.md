---
name: sporttrax-new-resource
description: Scaffold a new API resource in the CLI — command(s), MCP tool(s), tests on both surfaces, and docs — conforming to every codified standard by construction. Use when adding CLI support for an API resource (result, athlete, team, event, ...).
---

# Scaffold a new SportTrax resource

Given a resource (e.g. `result`, `athlete`), build its full CLI presence in
ONE change. `meet` (internal/cli/meet.go, internal/mcp/server.go) is the
canonical example — mirror it. Read CLAUDE.md first; its standards govern.

## 0. Establish the server contract first

Before writing any Go, pin down what the API actually exposes — from its
documentation, or by exercising the endpoints against a real instance.
Never infer a contract from what the CLI already does. Establish:

- **Endpoints** — which exist for this resource (index? show? both?)
- **Field names** — exact, verbatim. The CLI typed struct and the MCP
  tool description must mirror the server's resource fields, not a
  paraphrase of them
- **Filters** — which are allowed, their matching semantics
  (exact/partial), allowed sorts, and any required anchor filter (results
  REQUIRE meet_id OR athlete_id OR team_id)
- **Validation rules** — anything the server requires together or
  constrains to an enum must be enforced client-side BEFORE any request,
  on both surfaces, with tests asserting zero requests were made
- **Enum values** — copy them from the server's actual enum; never guess.
  A value that looks plausible but is wrong does not error, it silently
  matches nothing, and a client-side list narrower than the server's
  locks users out of real data

## 1. Typed struct — internal/api/resources.go

Add the resource struct mirroring the server resource's fields exactly
(snake_case json tags). Used for table/detail rendering only; --json and
MCP results are raw passthrough via List/GetRaw.

## 2. Command — internal/cli/<resource>.go

- Singular noun, `list` and/or `view` verbs matching available endpoints
- One flag per server filter, help text enumerating closed value sets;
  `RegisterFlagCompletionFunc` for enums; `--limit` (default 30, "all"
  warns) on list commands via the shared parseLimit
- Anchor/dependency rules validated before any request with an actionable
  error naming the flags
- Table columns answer "which one do I want" (ID, name, discriminators,
  when/where, ~100 cols); view shows the full typed record via
  ui.KeyValues; labels follow the label standard (mechanical humanization,
  sentence case, headers ALL CAPS); booleans true/false
- `--json` = raw items, `ui.JSON`; never curated

## 3. MCP tool(s) — internal/mcp/server.go

- `list_<resource>s` / `get_<resource>` registered in NewServer
- Args struct with jsonschema descriptions; enums/patterns tightened via
  an explicit InputSchema (see listMeetsSchema); same pre-request
  validation as the command
- Description states what it returns (field list) and when to use it
- Add the tool names to the `mcp` command's help text (Tools: line)

## 4. Tests — both surfaces, no exceptions

- internal/cli/<resource>_test.go via the harness (runCommand + stub
  serving the real envelope): TSV rows, --json passthrough (include a
  future_field in stubs and assert it survives), limit-stops-paging with
  request counts, filter mapping, dependency-rule errors with zero
  requests, not-logged-in → auth.ErrNotFound
- internal/mcp/server_test.go: registration with schema, at least one
  CallTool asserting filter mapping + passthrough, invalid enum rejected
  before any request

## 5. Finish

- `make docs` (regenerates command reference)
- README: add a short example block to the resource section
- Run the sporttrax-precommit-audit skill; fix findings before committing
- Commit both surfaces together — a command without its MCP tool is an
  incomplete change (CLAUDE.md parity rule)
