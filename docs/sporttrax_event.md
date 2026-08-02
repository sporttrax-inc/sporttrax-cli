## sporttrax event

Work with events

### Synopsis

Browse the gendered events results are recorded against.

Each event instantiates a base event for one gender — "Female 100m
Hurdles" is the female event for the 100-meter-hurdles base event. See
`sporttrax base-event` for the ungendered catalog.

### Options

```
  -h, --help   help for event
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
* [sporttrax event list](sporttrax_event_list.md)	 - List events
* [sporttrax event view](sporttrax_event_view.md)	 - View a single event

