# Contributing

Thanks for helping out. This is a small, opinionated CLI, and most of its
rules exist because breaking them produces a subtly wrong tool rather than
a broken one. The two worth reading before you write code are **MCP
parity** and **generated files** — CI enforces both, and neither is
guessable from the surrounding code.

Security issues do **not** belong in a pull request or a public issue. See
[SECURITY.md](SECURITY.md).

## Getting set up

Requires [Go](https://go.dev) 1.26 or newer.

```sh
make setup    # activates the committed pre-commit hook
make build    # bin/sporttrax
make test
```

Two tools the checks need:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Pin golangci-lint to the version CI uses (**v2.12.2**). The config declares
`version: "2"`, and a v1 binary refuses to run against it at all.

## The two rules that will fail your PR

### 1. MCP parity

`sporttrax mcp` serves the same data as the commands, over the Model
Context Protocol. **A data capability must land on both surfaces in the
same change.** A new `list`/`view` command needs its sibling tool; a new
filter flag needs the matching schema field; changed semantics need an
updated tool description.

A command without its tool is an incomplete change, not a follow-up.

```
internal/cli/<resource>.go   → the command
internal/mcp/<resource>s.go  → list_<resource> / get_<resource>
```

Both need tests. `internal/mcp` has an in-memory MCP client harness —
register the tool, then assert at least one `CallTool` maps its filters
and passes records through verbatim.

### 2. `docs/` and `THIRD_PARTY_NOTICES.md` are generated

Never edit either by hand. CI regenerates them and fails on any diff.

```sh
make docs      # after any command, flag, or help-text change
make notices   # after any dependency change
```

Command help text (`Short`, `Long`, `Example`) is the documentation source
of truth — `docs/` is produced from it.

**Dependabot PRs that bump Go modules will fail this check**, because the
bot cannot run `make notices`. That is the check working: the binary is
statically linked, so a dependency change alters which licenses ship with
it. Run `make notices` on the PR branch and push the result. Updates are
grouped weekly so this is one regeneration, not one per module.

## Conventions

The full set lives in [CLAUDE.md](CLAUDE.md). The ones that come up most:

- **Design follows `gh`, `aws`, and `stripe`.** Deviating is allowed, but
  it has to be a stated decision recorded in CLAUDE.md, not an accident.
- **Singular nouns, noun/verb commands** — `meet list`, not `meets list`.
- **Labels are mechanical humanizations of the JSON field name.** Strip
  `is_`/`has_` and `_at`, underscores become spaces, sentence case,
  acronyms stay upper. `hs_graduation_year` → `HS graduation year`. Never
  invent vocabulary: a user has to be able to guess the `--json` field
  from any label. Table headers are the exception — ALL CAPS.
- **`--json` is the data contract and is never curated.** It passes the
  server's records through verbatim, so new API fields reach consumers
  without a CLI release. Typed structs exist for rendering only.
- **Server enums live in `internal/api/enums.go`**, read by both surfaces.
  Never restate the values inline — that is how `road` came to be a
  listable but unfilterable sport.
- **No silent no-ops.** A flag or argument a command cannot honor is an
  error. List commands are `cobra.NoArgs`; scope a flag to the commands
  that actually read it.
- **Validate before the request.** The API leaves most filters
  unvalidated, so a wrong value silently matches nothing. Enforce enums
  client-side, on both surfaces, with tests asserting zero requests were
  made.
- **Rendered output is sanitized; `--json` is not.** Names are
  user-submitted, so terminal escapes are stripped and CSV cells that a
  spreadsheet would run as a formula are neutralized.

## Before you open a PR

```sh
make lint      # golangci-lint, including gofumpt formatting
make test
make docs      # commit any regenerated files
make notices   # only if you touched dependencies
make vuln
```

`make lint` is stricter than `gofmt` — it includes **gofumpt**, so code
can pass `gofmt -l` and still fail CI. Run it.

The pre-commit hook (`make setup`) covers vet, tests, tidy, and docs
drift. CI additionally checks formatting, lint, notices drift, and scans
for vulnerabilities on the Go version `go.mod` targets.

In the PR itself, say **why** rather than what — the diff already shows
what. If you deviated from a convention above, say so explicitly; an
explained deviation is reviewable, an unexplained one reads as an
oversight.

## License

Contributions are licensed under [Apache-2.0](LICENSE), the same terms as
the project. Note that the license covers code only — the SportTrax name
and logo are not licensed under it, so a fork must not use them to
identify itself.
