## sporttrax event list

List events

```
sporttrax event list [flags]
```

### Examples

```
  sporttrax event list
  sporttrax event list --gender female
  sporttrax event list --base-event 1
  sporttrax event list --limit all --csv > events.csv
```

### Options

```
      --base-event int    filter by base event ID
      --gender string     filter by gender: male, female, mixed
  -h, --help              help for list
  -L, --limit string      maximum events to fetch, or "all" (default "30")
      --multi-event int   filter by multi event ID
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

* [sporttrax event](sporttrax_event.md)	 - Work with events

