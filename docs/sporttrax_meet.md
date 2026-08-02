## sporttrax meet

Work with meets

### Synopsis

Browse published meets.

Meets are the top of the tree: a meet's ID anchors `sporttrax result list`,
which is where athlete and team IDs come from in turn.

### Options

```
  -h, --help   help for meet
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
* [sporttrax meet list](sporttrax_meet_list.md)	 - List published meets
* [sporttrax meet view](sporttrax_meet_view.md)	 - View a single meet

