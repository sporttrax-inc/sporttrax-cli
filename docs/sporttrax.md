## sporttrax

SportTrax command-line interface

### Synopsis

sporttrax is a fast, cross-platform CLI for the SportTrax APIs.

### Options

```
      --api-url string   base URL override for the SportTrax API, https only (http is allowed for localhost); takes precedence over --env (env: SPORTTRAX_API_URL)
      --csv              output as CSV with a header row (for spreadsheets)
      --env string       named environment: production, staging, testing, or one from config.yaml (env: SPORTTRAX_ENV) (default "production")
  -h, --help             help for sporttrax
      --insecure         skip TLS certificate verification (self-signed dev instances only; refused for the stock environments)
      --json             output as JSON instead of formatted text
      --no-color         disable color and styling (NO_COLOR is also honored)
      --verbose          log API requests to stderr (env: SPORTTRAX_DEBUG)
```

### SEE ALSO

* [sporttrax athlete](sporttrax_athlete.md)	 - Work with athletes
* [sporttrax auth](sporttrax_auth.md)	 - Authenticate with the SportTrax API
* [sporttrax base-event](sporttrax_base-event.md)	 - Work with base events
* [sporttrax env](sporttrax_env.md)	 - Inspect SportTrax environments
* [sporttrax event](sporttrax_event.md)	 - Work with events
* [sporttrax mcp](sporttrax_mcp.md)	 - Run an MCP server exposing SportTrax data as AI tools
* [sporttrax meet](sporttrax_meet.md)	 - Work with meets
* [sporttrax result](sporttrax_result.md)	 - Work with results
* [sporttrax team](sporttrax_team.md)	 - Work with teams
* [sporttrax version](sporttrax_version.md)	 - Print the sporttrax version

