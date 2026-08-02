## sporttrax athlete

Work with athletes

### Synopsis

Look up an athlete by ID.

The API exposes athletes individually, not as a list, so there is no
athlete search. Athlete IDs come from results: `sporttrax result list
--meet <id>` carries an athlete_id on every row.

### Options

```
  -h, --help   help for athlete
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
* [sporttrax athlete view](sporttrax_athlete_view.md)	 - View a single athlete

