## sporttrax meet list

List published meets

```
sporttrax meet list [flags]
```

### Examples

```
  sporttrax meet list
  sporttrax meet list --sport track --from 2026-06-01
  sporttrax meet list --state ID --city Boise
  sporttrax meet list --name "state" --limit 100
  sporttrax meet list --json | jq '.[].name'
```

### Options

```
      --city string    filter by venue city (requires --state)
      --from string    meets starting on or after this date (YYYY-MM-DD)
  -h, --help           help for list
  -L, --limit string   maximum meets to fetch, or "all" (default "30")
      --name string    filter by name (partial match)
      --sport string   filter by sport: track, xc, road
      --state string   filter by two-letter venue state code, e.g. ID
      --to string      meets starting on or before this date (YYYY-MM-DD)
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

* [sporttrax meet](sporttrax_meet.md)	 - Work with meets

