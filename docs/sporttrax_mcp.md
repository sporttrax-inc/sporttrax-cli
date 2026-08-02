## sporttrax mcp

Run an MCP server exposing SportTrax data as AI tools

### Synopsis

Run a Model Context Protocol server over stdio, exposing the
SportTrax public API as structured tools for AI assistants. Authentication
and environment selection work exactly like every other command: the stored
token for --env (or SPORTTRAX_API_TOKEN) is used.

Register with Claude Code:

  claude mcp add sporttrax -- sporttrax mcp

Or in any MCP host's config:

  {"command": "sporttrax", "args": ["mcp"]}

Point it at another environment with --env, e.g.:

  claude mcp add sporttrax-testing -- sporttrax --env testing mcp

Tools: whoami, list_meets, get_meet, list_results, get_result,
list_events, get_event, list_base_events, get_base_event, get_athlete,
get_team

```
sporttrax mcp [flags]
```

### Options

```
  -h, --help   help for mcp
```

### Options inherited from parent commands

```
      --api-url string   base URL override for the SportTrax API, https only (http is allowed for localhost); takes precedence over --env (env: SPORTTRAX_API_URL)
      --csv              output as CSV with a header row (for spreadsheets)
      --env string       named environment: production, staging, testing, or one from config.yaml (env: SPORTTRAX_ENV) (default "production")
      --insecure         skip TLS certificate verification (self-signed dev instances only; refused for the stock environments)
      --json             output as JSON instead of formatted text
      --no-color         disable color and styling (NO_COLOR is also honored)
      --verbose          log API requests to stderr (env: SPORTTRAX_DEBUG)
```

### SEE ALSO

* [sporttrax](sporttrax.md)	 - SportTrax command-line interface

