# sporttrax-cli

Public, cross-platform Go CLI (cobra) for the SportTrax public API
(`/public-api/v1`). Binary is `sporttrax`; repo keeps the `-cli` suffix.

## Guiding rule: follow the leaders

Model every design decision on the most widely adopted CLIs — `gh`,
`aws` (awscli), and `stripe` are the reference set. When existing code or a
requested change strays from what those tools do (command structure, flag
naming, auth flows, config/token storage, output formats, exit codes,
release/packaging), **call out the deviation explicitly** before or while
implementing, and say what the reference tools do instead. Deviating is
allowed when there's a SportTrax-specific reason, but it must be a stated
decision, not an accident.

Conventions already adopted (keep consistent with them):

- noun/verb command tree (`auth login`, `env list`), cobra + persistent flags
- PAT auth: paste-token login validated against the API; OS keychain first,
  `hosts.json` (0600, re-narrowed on read/write) fallback;
  `SPORTTRAX_API_TOKEN` env overrides all
- named environments (production/staging/testing built in; custom ones in
  config) selected via `--env`/`SPORTTRAX_ENV`; `--api-url`/`SPORTTRAX_API_URL`
  win as raw overrides; tokens keyed per API host
- **token egress rule**: a token only reaches a host the user vouched for.
  Stored tokens are host-keyed by construction; `SPORTTRAX_API_TOKEN` is
  sent only to a stock environment, a `config.yaml` environment, or
  localhost — never to an arbitrary `--api-url` (gh's GH_TOKEN /
  GH_ENTERPRISE_TOKEN split, adapted). API URLs must be https, except
  localhost. Disabling TLS verification is quiet only for localhost, warns
  on every command for any other host, and is refused for the stock
  environments
- config lives in `~/.config/sporttrax-cli/` on macOS and Linux
  (`$XDG_CONFIG_HOME` honored), `%AppData%\sporttrax-cli` on Windows
- only public identifiers ship in the repo/binary (Pusher app keys, URLs);
  real secrets are user-provided or server-side — never embedded
- all API calls go through internal/api Client.Get: sets User-Agent and the
  X-SPORTTRAX-* client headers (DEVICE-TYPE "cli", CLI-VERSION, CLIENT-NOW —
  the same pattern every SportTrax client follows; no persistent device ID
  by design), retries 429 honoring Retry-After (bounded 90s) and
  transient 5xx/network errors, maps status codes to actionable messages,
  and detects the X-SPORTTRAX-DOWN maintenance header
- exit codes are contract: 0 success, 1 error, 2 cancelled (Ctrl-C via
  signal.NotifyContext), 4 auth failure
- `--verbose`/SPORTTRAX_DEBUG logs requests to stderr; `--no-color`/NO_COLOR
  disables styling
- output via internal/ui: rounded bordered tables (dim borders, bold
  headers; square-corner fallback on Windows) on a TTY — a deliberate
  deviation from gh's plain columns, closer to `aws --output table` —
  header-less raw-value TSV when piped, global `--json` flag on every data
  command; interactive prompts/spinners via charmbracelet huh
- global `--csv` flag (RFC 4180, **with** a header row — it is opened in a
  spreadsheet, where labelled columns are the point; the header is the one
  place CSV and piped TSV deliberately differ). Mutually exclusive with
  `--json`. Detail views become `field,value` rows. Cells a spreadsheet
  would run as a formula (leading `=`, `+`, `@`, or non-numeric `-`) are
  apostrophe-prefixed — the same defense as terminal-escape sanitizing,
  aimed at the program that opens the file. Negative numbers are left
  numeric. A boolean flag rather than aws's `--output csv` for consistency
  with the `--json` already established here
- **label standard**: display labels are mechanical humanizations of the
  JSON field name, never new vocabulary — strip boolean prefixes (is_/has_)
  and timestamp suffixes (_at/_starting_at), underscores → spaces, sentence
  case ("First session"), acronyms stay upper (ID, URL). Table headers are
  the exception: ALL CAPS. Composed display values (e.g. Venue) are labeled
  by their parent field. A user must be able to guess the --json field from
  any label
- **no silent no-ops**: a flag or argument the command cannot honor is an
  error, never ignored. List commands are `cobra.NoArgs`; `--units` is
  scoped to `result list` (the only place a mark renders as one value);
  `--json`/`--csv` are honored by every command that outputs data,
  `version` included
- **server enums live in internal/api/enums.go**, read by both surfaces —
  commands for help text, completions, and pre-request validation, MCP for
  input schemas. Never restate the values inline: the `road` gap happened
  because one list was copied in five places
- **validation parity**: a filter enforced on one surface is enforced on
  the other, client-side, before any request. The API leaves most filters
  unvalidated (a wrong value silently matches nothing), and even where it
  does validate, a local error beats a round trip
- **rendered text is sanitized**: server data reaches a terminal only
  through internal/ui, which strips escape sequences and control
  characters (names are user-submitted, so they can carry screen-clearing
  or output-forging escapes). `--json` and the MCP tools stay verbatim —
  they are the data contract and nothing interprets them as a terminal
- **value standard**: booleans render true/false (the server's vocabulary);
  piped output carries the same values as the TTY minus styling —
  value-level divergence is allowed only where the TTY form is lossy or
  meaningless to a machine (e.g. masked secrets are unmasked when piped).
  --json is the data contract; TSV is a convenience
- **table columns**: list rows answer "which one do I want" — ID, name,
  discriminators, when/where; fit ~100 cols. Detail views show the full
  typed record. --json is never curated
- **fields kept out of rendered views are omitted from the typed struct**,
  not skipped at render time — the struct exists for rendering, so a field
  it lacks simply has no rendered form, and "detail views show the full
  typed record" stays literally true. Base events do this with the TFRRS
  and Hy-Tek codes: integration identifiers for other systems, not
  something a results table should show. They still reach consumers
  untouched via --json and the MCP tools, which are raw passthrough
- `--version` and `version` both work and print the same line
- releases: goreleaser, static binaries for darwin/linux/windows ×
  amd64/arm64; version injected via ldflags (`make build`, `make cross`)

## Audience & north star

Third parties will build on the SportTrax API — real-time dashboards,
integrations, tooling. The CLI is a primary integration surface, not just a
human tool. Consequences:

- **Programmatic consumption is first-class.** Every data command must be
  cleanly machine-readable: `--json` everywhere, header-less TSV when piped,
  stable field names (treat JSON output shape as a semver-relevant API),
  meaningful exit codes, errors to stderr only.
- **Streaming commands emit NDJSON under `--json`** (one JSON object per
  line per event) so `meet watch --json | ...` can feed dashboards and
  pipelines directly.
- **MCP parity is mandatory (top priority).** `sporttrax mcp` serves the
  API as MCP tools over stdio (the gh/stripe pattern), sharing the typed
  internal/api client with command handlers. Every change that adds or
  alters a data capability MUST, in the same change: register/update the
  sibling MCP tool in internal/mcp, with an input schema and a description
  an AI can use without trial-and-error (state what it returns and when to
  use it); list tools return {data, count, has_more} with verbatim records
  inside data; update the tool list in the `mcp` command help; and test the
  tool through the in-memory MCP client harness (internal/mcp tests). A
  data command without its MCP tool is an incomplete change — the
  precommit audit fails it.

## Decisions (settled — do not relitigate, flag conflicts)

- **Language: Go.** A single static binary with no runtime dependency is a
  hard requirement for public distribution and startup speed; interpreted
  alternatives were considered and rejected on that basis.
- **Nouns are singular** — `meet list`, `result list`, `athlete view`
  (gh-style). Never plural.
- **Hyphenated nouns are allowed when the API uses one** — `base-event`
  mirrors the `base-events` endpoint and the `base_event_id` field that
  `result list --base-event` already filters on. A deviation from gh,
  which never hyphenates a noun; taken because the label standard's
  guessability rule outranks it — inventing `event-type` would leave the
  command saying one thing and `--json` saying another
- **No colon aliases.** `meet:watch` is not a command; space-separated only.
- **Pagination: gh-style `--limit N`** — default ~30 items, larger limits
  transparently follow cursors, `--limit all` depletes with a rate-limit
  warning (non-admins: 15 req/min, 1000/day). No raw cursor flags.
- **JSON: plain `--json` only.** No --jq/--template/field selection unless
  demanded later; users pipe to jq.

## Not built yet (decided, unimplemented — do not describe as shipped)

Everything above describes the CLI as it is. These are settled decisions
awaiting work; treat a reference to one as a plan, not a feature.

- **Distribution beyond GitHub Releases.** Releases are built by
  goreleaser from a `v*` tag (.github/workflows/release.yml) and are the
  only channel that exists. A Homebrew tap
  (`sporttrax-inc/homebrew-tap`, needs a `homebrew_casks` block) and a
  curl installer script are planned; no Scoop/deb/rpm for now. `go
  install` works today by virtue of the repo being public.
- **Update notification: notify-only** — gh-style check of the releases
  feed, cached ≤1/24h, skipped when non-TTY/CI; prints an upgrade hint.
  No self-update command. Nothing is implemented; there is no version
  check in the binary at all.
- **Streaming.** `meet watch` emitting NDJSON under `--json`. Pusher
  connection details are already carried per environment for it, but no
  streaming code exists — and the server's broadcast channels are private
  and session-authorized, so a public-API token cannot subscribe to them
  yet. That is a server-side prerequisite, not a CLI task.

## Development

- one-time: `make setup` (activates .githooks/pre-commit: fmt, vet,
  golangci-lint, tests, tidy, docs drift — mechanical failures are
  uncommittable) and `brew install golangci-lint govulncheck`
- `make build` / `make test` / `make lint` (golangci-lint) / `make vuln`
  (govulncheck) / `make cross`; CI (.github/workflows/ci.yml) runs the same
  checks plus govulncheck on every push and PR
- new API resources: use the `sporttrax-new-resource` skill — it scaffolds
  command + MCP tool + tests + docs conforming to every standard
- `//nolint` always carries an inline reason
- run tests before committing; smoke-test commands against a stub server
  where practical (see auth work for the pattern)
- run the `sporttrax-precommit-audit` skill before committing non-trivial
  changes — it checks standards, security, performance, and formatting
- docs/ is generated from the cobra definitions (`make docs`) — command
  help text (Short/Long/Example) is the documentation source of truth;
  never edit docs/ by hand, regenerate after any command change
- THIRD_PARTY_NOTICES.md is generated from the linked module graph
  (`make notices`) — the binary is statically linked, so the MIT/BSD
  notices of everything compiled into it must ship with it. Any
  dependency change requires regenerating; CI fails on drift, and the
  generator fails rather than silently omitting a module it can't find a
  license for
