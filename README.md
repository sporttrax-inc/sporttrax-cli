# sporttrax-cli

`sporttrax` is a fast, cross-platform CLI for the SportTrax public API. It
ships as a single static binary for macOS, Linux, and Windows (amd64 and
arm64) with no runtime dependencies.

It is built for getting data out. Every data command speaks `--csv` for
spreadsheets and `--json` for scripts, piped output is plain TSV, and
`sporttrax mcp` exposes the same data to AI assistants. If you run meets,
time them, or build dashboards from their results, this is the supported
way to reach that data.

## Requirements

A SportTrax account and a personal access token with the `public-api`
permission — create one in the web UI under **API Tokens**. Nothing else:
the binary bundles everything it needs.

## Install

**Homebrew** (macOS and Linux):

```sh
brew install sporttrax-inc/tap/sporttrax
brew upgrade sporttrax
```

**Install script** (macOS and Linux) — downloads the latest release and
verifies its checksum before installing:

```sh
curl -fsSL https://raw.githubusercontent.com/sporttrax-inc/sporttrax-cli/main/install.sh | sh
```

It installs to `/usr/local/bin` when that is writable and `~/.local/bin`
otherwise, telling you if that directory is not on your `PATH`. Set
`SPORTTRAX_INSTALL_DIR` to choose, or `SPORTTRAX_VERSION` to pin a
release. Reading [install.sh](install.sh) before piping it to a shell is
a reasonable habit.

**Download a release.** Pick your platform from the
[latest release](https://github.com/sporttrax-inc/sporttrax-cli/releases/latest).
Archives are named `sporttrax_<version>_<os>_<arch>` — `.tar.gz`, or
`.zip` on Windows — and contain the binary alongside its license files:

```sh
# macOS, Apple silicon — swap darwin_arm64 for your platform
VERSION=0.1.0
curl -fsSL -o sporttrax.tar.gz \
  "https://github.com/sporttrax-inc/sporttrax-cli/releases/download/v${VERSION}/sporttrax_${VERSION}_darwin_arm64.tar.gz"
tar xzf sporttrax.tar.gz sporttrax
sudo mv sporttrax /usr/local/bin/
```

Every release ships a `checksums.txt` if you want to verify the download.

**With Go** (1.26 or newer):

```sh
go install github.com/sporttrax-inc/sporttrax-cli/cmd/sporttrax@latest
```

The binary lands in `$(go env GOPATH)/bin`, which needs to be on your
`PATH`.

**From source**: clone this repo and run `make build` — the binary appears
at `bin/sporttrax`.

Confirm it worked with `sporttrax version` (or `sporttrax --version` —
both print the same line).

### If your operating system warns you

Releases are not yet code-signed. Homebrew, the install script, and `go
install` are unaffected — none of them mark the binary as
browser-downloaded. Downloading an archive **through a browser** does mark
it, and then:

- **macOS** refuses to run it, saying it cannot check the developer.
  Approve it in **System Settings → Privacy & Security**, or strip the
  mark yourself: `xattr -d com.apple.quarantine ./sporttrax`
- **Windows** may show a SmartScreen warning; **More info → Run anyway**.

Every release publishes a `checksums.txt`, so you can confirm you have the
bytes we built regardless of what your operating system says about them.

## Quickstart

```sh
sporttrax auth login                                # paste your token, once
sporttrax meet list                                 # 30 most recent meets
sporttrax result list --meet 4821                   # results for one meet
sporttrax result list --meet 4821 --csv > results.csv
```

## Authentication

The CLI authenticates with a SportTrax personal access token (create one in
the web UI under **API Tokens** with the `public-api` permission checked):

```sh
sporttrax auth login       # opens the token page, prompts for the token
sporttrax auth status      # who you are, masked token, live validity check
sporttrax auth logout      # remove the stored token
```

Login and status validate the token against the API and report the
authenticated user, e.g. `Logged in to sporttrax.com as Coach, Salem
Hills (admin)`.

Tokens are validated at login and stored in the OS keychain (macOS Keychain,
Windows Credential Manager, Linux Secret Service), falling back to
`hosts.json` (0600) in the config directory on headless machines. For CI, set
`SPORTTRAX_API_TOKEN` — it takes precedence over stored tokens. Non-interactive
login: `echo $TOKEN | sporttrax auth login --with-token`.

Stored tokens are keyed by API host and only ever go to that host.
`SPORTTRAX_API_TOKEN` has no such binding, so it is only sent to hosts the
CLI already knows — nothing in a CI job's environment can redirect your
token to someone else's server. Requests are always made over `https`.

## Meets

```sh
sporttrax meet list                                    # 30 most recent meets
sporttrax meet list --sport track --from 2026-06-01 --to 2026-07-01
sporttrax meet list --sport road                       # track, xc, road
sporttrax meet list --state ID                         # meets at Idaho venues
sporttrax meet list --state ID --city Boise            # city requires --state
sporttrax meet list --name "state" --limit 100         # follows pages as needed
sporttrax meet list --limit all                        # deplete (rate-limit aware)
sporttrax meet view 4821
sporttrax meet list --json | jq '.[].name'
```

`--limit` (default 30) transparently follows the API's cursor pagination;
`--json` returns the server's records verbatim, so new API fields appear
without a CLI update.

Filters that map to a server enum (`--sport`, and on results `--gender`,
`--level`, `--round`, `--sort`) are validated before any request is made,
since the API silently matches nothing for an unrecognized value. Each
command's `--help` lists the accepted values.

Marks in `result list` follow `sporttrax result list --units english|metric`,
or a `units:` line at the top of `~/.config/sporttrax-cli/config.yaml`
(`%AppData%\sporttrax-cli` on Windows) to set it once. Markless results
(DNF, DQ, …) show their status code. `result view` lists the english and
metric forms as their own fields, and JSON always carries both — units
affect display only.

## Results

Results require at least one anchor: `--meet`, `--athlete`, or `--team`.

```sh
sporttrax result list --meet 4821
sporttrax result list --athlete 992 --sport track --round finals
sporttrax result list --meet 4821 --sport road --gender female
sporttrax result list --team 55 --from 2026-06-01 --sort -at
sporttrax result list --meet 4821 --official true --json | jq '.[].mark.display'
sporttrax result view 120345
```

## Athletes and teams

The API exposes athletes and teams individually rather than as lists, so
there is no search. IDs come from results — every result row carries an
`athlete_id` and `team_id`:

```sh
sporttrax athlete view 992
sporttrax team view 55 --json
```

## Events

An **event** is one gender's instance of a **base event**: "Female 100m
Hurdles" and "Male 100m Hurdles" are two events sharing one
`100-meter-hurdles` base event. The base event carries the sport, mark
type, distance, and classification (track/field, relay, hurdles, throw,
…), which is what makes results groupable.

```sh
sporttrax event list --gender female
sporttrax event list --base-event 1
sporttrax event view 1

sporttrax base-event list --sport track
sporttrax base-event list --mark-type time --limit all
sporttrax base-event view 1
sporttrax base-event list --csv > event-catalog.csv
```

Events cannot be filtered by meet — the API offers no such filter. To see
which events a meet ran, list its results.

## Output

Data commands print aligned tables on a terminal and header-less
tab-separated lines when piped (grep/cut friendly). When a list is
truncated at `--limit`, a footer says so (terminal only — piped output
stays clean). Pass `--json` anywhere for structured output:

```sh
sporttrax result list --meet 4821 --json | jq '.[].mark.display'
sporttrax auth status --json
```

### Spreadsheets: `--csv`

`--csv` writes RFC 4180 CSV with a header row, ready to open in Excel,
Numbers, or Google Sheets — the quickest path from a meet to a chart:

```sh
sporttrax result list --meet 4821 --limit all --csv > results.csv
sporttrax meet list --sport road --csv > road-meets.csv
sporttrax meet view 4821 --csv          # detail views become field,value rows
```

Unlike the piped TSV form, CSV keeps its header: it is written to be
opened in a spreadsheet, where labelled columns are the point. `--csv` and
`--json` are mutually exclusive — `--json` is the data contract, CSV is a
convenience for people building dashboards and reports.

Values that a spreadsheet would interpret as a formula (a cell starting
with `=`, `+`, `@`, or a non-numeric `-`) are prefixed with an apostrophe
so they stay literal text. Meet and athlete names are user-submitted, and
a formula in one of them would otherwise run when the file is opened.
Negative numbers such as wind readings are left alone.

## MCP (AI tools)

`sporttrax mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io)
server over stdio, exposing SportTrax data as structured tools
(`list_meets`, `get_meet`, `list_results`, `list_events`, `get_athlete`, …)
that AI assistants can call directly — no scraping, no shell access. It
uses the token you already logged in with, so setup is two steps:

```sh
# 1. Install the CLI and log in once
sporttrax auth login

# 2. Register it with your AI tool (examples below)
```

After registering, just ask questions in your AI tool — "What meets
happened in Idaho this month?", "Pull up meet 4821 and summarize it" — and
it calls the SportTrax tools with your credentials.

### Claude Code

```sh
claude mcp add sporttrax -- sporttrax mcp
```

### Claude Desktop

Add to `claude_desktop_config.json` (Settings → Developer → Edit Config):

```json
{
  "mcpServers": {
    "sporttrax": { "command": "sporttrax", "args": ["mcp"] }
  }
}
```

### Cursor

Add the same block to `~/.cursor/mcp.json` (or per-project
`.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "sporttrax": { "command": "sporttrax", "args": ["mcp"] }
  }
}
```

### VS Code (GitHub Copilot)

Add to `.vscode/mcp.json` in your workspace:

```json
{
  "servers": {
    "sporttrax": { "type": "stdio", "command": "sporttrax", "args": ["mcp"] }
  }
}
```

### Other hosts

Any MCP-capable tool (Windsurf, Codex CLI, Gemini CLI, …) works with the
same stdio shape: command `sporttrax`, args `["mcp"]`.

Troubleshooting: run `sporttrax mcp` by hand — if it prints an auth error,
log in first, and `sporttrax auth status` will confirm the token is valid.

## Documentation

The full command reference lives in [docs/sporttrax.md](docs/sporttrax.md),
generated from the command definitions — run `make docs` after changing any
command; never edit `docs/` by hand.

## Development

Requires [Go](https://go.dev) 1.26+, plus `golangci-lint` and `govulncheck`
(`brew install golangci-lint govulncheck`).

```sh
make setup          # one-time: activate the committed git pre-commit hooks
make build          # build bin/sporttrax for your machine
make test           # run tests
make lint           # golangci-lint (staticcheck, errcheck, gosec, ...)
make vuln           # govulncheck against the Go vulnerability database
make notices        # regenerate THIRD_PARTY_NOTICES.md from the module graph
make cross          # build all six platform binaries into dist/
```

## Releasing

Pushing a `v*` tag runs [.github/workflows/release.yml](.github/workflows/release.yml),
which tests, scans for vulnerabilities, and publishes the archives with
[goreleaser](https://goreleaser.com):

```sh
git tag v0.1.0 && git push origin v0.1.0
```

Releasing from CI rather than a laptop is deliberate — the binaries are
built on the Go toolchain `go.mod` resolves to, so a locally installed
version can't quietly become the one you ship.

For a local dry run, artifacts land in `dist/` and nothing is published:

```sh
goreleaser release --snapshot --clean
```

## Layout

```
cmd/sporttrax/      entrypoint
internal/cli/       cobra command definitions
internal/api/       SportTrax API client
internal/version/   build metadata injected via ldflags
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Two rules are worth knowing before
you start: every data capability ships on both the CLI and MCP surfaces in
the same change, and `docs/` and `THIRD_PARTY_NOTICES.md` are generated —
CI fails on drift in either.

## Security

To report a security issue, see [SECURITY.md](SECURITY.md). Please do not
open a public issue for vulnerabilities.

## License

Copyright 2026 SportTrax, Inc.

Licensed under the Apache License, Version 2.0 — see [LICENSE](LICENSE).
You may obtain a copy at
<https://www.apache.org/licenses/LICENSE-2.0>.

The binary is statically linked and includes third-party Go modules under
their own licenses, reproduced in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and shipped inside every
release archive. Regenerate with `make notices` after any dependency
change.

**Trademarks.** The license covers the source code only. "SportTrax", the
SportTrax name, logo, and other brand assets are trademarks of SportTrax,
Inc. and are **not** licensed under Apache-2.0 (see section 6 of the
license). Forks and derivative works must not use the SportTrax name,
logo, or a confusingly similar mark to identify themselves, imply
endorsement, or name their distributed binaries.
