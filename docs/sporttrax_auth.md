## sporttrax auth

Authenticate with the SportTrax API

### Synopsis

Manage authentication with the SportTrax API.

Tokens are stored per environment (keyed by API host), so you can be logged
in to production, staging, and testing at the same time. Target an
environment with --env (or a raw URL with --api-url):

  sporttrax auth login                     # production
  sporttrax --env testing auth login       # stock environment
  sporttrax --env ultra auth login         # custom environment from config.yaml
  sporttrax env list                       # see all environments

### Options

```
  -h, --help   help for auth
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
* [sporttrax auth login](sporttrax_auth_login.md)	 - Log in with a personal access token
* [sporttrax auth logout](sporttrax_auth_logout.md)	 - Remove the stored token for the selected environment
* [sporttrax auth status](sporttrax_auth_status.md)	 - Show authentication status for the selected environment

