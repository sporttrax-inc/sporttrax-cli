## sporttrax result list

List results for a meet, athlete, or team

### Synopsis

List results. At least one anchor filter is required: --meet,
--athlete, or --team.

```
sporttrax result list [flags]
```

### Examples

```
  sporttrax result list --meet 4821
  sporttrax result list --athlete 992 --sport track --round finals
  sporttrax result list --team 55 --from 2026-06-01 --sort -at
  sporttrax result list --meet 4821 --official true --json | jq '.[].place'
```

### Options

```
      --athlete int       anchor: filter by athlete ID
      --base-event int    filter by base event ID
      --event int         filter by event ID
      --from string       results on or after this date (YYYY-MM-DD)
      --gender string     filter by gender: male, female, mixed
  -h, --help              help for list
      --level string      filter by level: professional, college, high_school, middle_school, elementary, unattached, hs_unified, club
  -L, --limit string      maximum results to fetch, or "all" (default "30")
      --meet int          anchor: filter by meet ID
      --official string   filter by official status: true, false
      --relay string      filter by relay results: true, false
      --round string      filter by round: finals, semi_finals, quarter_finals, prelims
      --sort string       sort order: at, -at, place, -place, id, -id (default id)
      --sport string      filter by sport: track, xc, road
      --team int          anchor: filter by team ID
      --to string         results on or before this date (YYYY-MM-DD)
      --units string      unit system for displayed marks: english, metric (default: "units" in config.yaml, else the mark's native form)
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

* [sporttrax result](sporttrax_result.md)	 - Work with results

