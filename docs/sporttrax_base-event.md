## sporttrax base-event

Work with base events

### Synopsis

Browse the ungendered event catalog.

A base event is the event itself — 100-meter-hurdles, shot put, 5k — with
its sport, mark type, and classification. `sporttrax event` lists the
gendered events that instantiate these.

### Options

```
  -h, --help   help for base-event
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
* [sporttrax base-event list](sporttrax_base-event_list.md)	 - List base events
* [sporttrax base-event view](sporttrax_base-event_view.md)	 - View a single base event

