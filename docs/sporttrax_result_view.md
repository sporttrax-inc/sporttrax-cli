## sporttrax result view

View a single result

```
sporttrax result view <id> [flags]
```

### Examples

```
  sporttrax result view 120345
  sporttrax result view 120345 --json
```

### Options

```
  -h, --help   help for view
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

