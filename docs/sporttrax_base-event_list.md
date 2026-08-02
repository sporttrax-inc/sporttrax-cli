## sporttrax base-event list

List base events

```
sporttrax base-event list [flags]
```

### Examples

```
  sporttrax base-event list
  sporttrax base-event list --sport track
  sporttrax base-event list --mark-type time --limit all
  sporttrax base-event list --csv > event-catalog.csv
```

### Options

```
  -h, --help               help for list
  -L, --limit string       maximum base events to fetch, or "all" (default "30")
      --mark-type string   filter by mark type: time, distance, score
      --sport string       filter by sport: track, xc, road
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

* [sporttrax base-event](sporttrax_base-event.md)	 - Work with base events

