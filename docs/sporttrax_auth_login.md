## sporttrax auth login

Log in with a personal access token

### Synopsis

Log in to a SportTrax environment with a personal access token.

Create a token in the web UI under API Tokens, making sure the "public-api"
permission is checked, then paste it at the prompt. The token is validated
against the API and stored in the OS keychain (or a config file if no
keychain is available).

```
sporttrax auth login [flags]
```

### Options

```
  -h, --help         help for login
      --no-browser   do not open the token creation page in a browser
      --with-token   read the token from standard input
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

* [sporttrax auth](sporttrax_auth.md)	 - Authenticate with the SportTrax API

