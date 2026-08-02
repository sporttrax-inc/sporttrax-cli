## sporttrax env list

List known environments and login state

### Synopsis

List the environments the CLI can target: the built-in ones
plus any defined or overridden in the config file. Select one with --env
or SPORTTRAX_ENV.

```
sporttrax env list [flags]
```

### Examples

```
  sporttrax env list
  sporttrax env list --json
```

### Options

```
  -h, --help   help for list
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

* [sporttrax env](sporttrax_env.md)	 - Inspect SportTrax environments

