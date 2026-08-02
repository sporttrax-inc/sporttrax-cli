---
name: sporttrax-precommit-audit
description: Audit outstanding changes before committing — project standards (CLAUDE.md), security, performance, formatting, and safety checks for the sporttrax CLI. Use before any commit, when asked to audit/validate the code, or as the final step of a feature.
---

# SportTrax pre-commit audit

Audit the outstanding changes (staged + unstaged + untracked) and report
PASS / WARN / FAIL per section below. FAIL blocks the commit; WARN needs a
stated justification. Read CLAUDE.md first — it is the source of truth for
standards; if this skill and CLAUDE.md disagree, CLAUDE.md wins.

## 1. Mechanical checks (run all; any failure = FAIL)

```sh
gofmt -l .                  # must print nothing
go vet ./...
golangci-lint run ./...     # staticcheck/errcheck/gosec/revive per .golangci.yml
go build ./...
go test ./...
go mod tidy -diff           # must produce no diff (or: tidy and diff go.mod/go.sum)
```

New `//nolint` directives in the diff require an inline reason and a
justification in the report (suppressions are findings too).

Also build one non-native target to catch platform-conditional code
(`runtime.GOOS` logic, path handling):

```sh
GOOS=windows GOARCH=amd64 go build ./...
```

Docs must not drift from the command definitions:

```sh
make docs && git diff --exit-code -- docs   # regenerate; must produce no diff
```

Run `govulncheck ./...` (installed via brew): reachable vulnerabilities are
FAIL (fix = upgrade the module or Go patch release), unreachable ones WARN.

## 2. Security & secrets (FAIL on any hit)

Scan the full diff (`git diff HEAD` plus untracked files) for:

- Tokens/secrets: SportTrax PATs (`\d+\|[A-Za-z0-9]{30,}`), any
  `*_SECRET` value, error-tracking or provider auth tokens, API secrets,
  private keys, `.env` file contents. **Allowlisted as public by design**:
  Pusher app *keys* (public client identifiers) and API base URLs.
- Personal/dev infrastructure that must never ship: tunnel URLs, VPN or
  mesh-network hostnames, and personal machine names. These belong in the
  user's local `config.yaml`, never in the repo.
- `InsecureSkipVerify` anywhere other than the single sanctioned site in
  `internal/api` gated behind the insecure option.
- New dependencies in go.mod: flag any addition for a quick
  provenance/necessity check (WARN, not FAIL).
- Hostnames from a local development setup (a developer's own domains or
  `.test`-style names): local dev targets are port-based URLs on
  localhost.

## 3. Project standards (from CLAUDE.md; violations = FAIL unless justified)

Check changed files for:

- **Command grammar**: singular nouns (`meet`, not `meets`), space-separated
  tree, no colon command names, noun/verb structure.
- **HTTP discipline**: every API call goes through `internal/api` Client —
  no `http.Get`/`http.NewRequest` in `internal/cli` or elsewhere (grep the
  diff). Client identification headers must not be bypassed.
- **Output discipline**: data commands render through `internal/ui`; every
  new data command supports `--json`; piped TSV carries raw values (no
  masking, no `—`/unicode placeholders); JSON field names mirror the
  server's PublicApi V1 resources and are snake_case. Streaming output must
  be NDJSON under `--json`.
- **Labels & values** (per CLAUDE.md standards): detail labels are
  mechanical humanizations of the JSON field name (strip is_/has_ and
  _at/_starting_at, underscores → spaces, sentence case, acronyms upper) —
  invented vocabulary is a FAIL; table headers ALL CAPS; booleans
  true/false; piped values identical to TTY values except lossy forms
  (masking); composed display values labeled by their parent field.
- **Errors/exit codes**: errors to stderr only; exit code contract preserved
  (0 ok / 1 error / 2 cancelled / 4 auth); user-facing errors actionable.
- **Reference-CLI rule**: anything deviating from gh/awscli/stripe behavior
  must be a recorded decision (CLAUDE.md), not an accident — flag new
  deviations.

## 4. MCP parity (FAIL on violation — top priority per CLAUDE.md)

AI tools consuming SportTrax via MCP are a core product goal. For every
data capability the diff adds or changes:

- A sibling MCP tool is registered/updated in `internal/mcp` in this same
  change (new list/view command → new tool; new filter flag → new schema
  field; changed semantics → updated description).
- The tool has an input schema (jsonschema tags) and a description that
  states what it returns and when to use it — good enough for an AI to
  call it correctly without trial-and-error. Vague descriptions are a FAIL.
- Resource payloads pass through verbatim (same contract as --json); list
  tools wrap them as {data, count, has_more} so AI callers can detect
  truncated windows.
- The tool is exercised through the in-memory MCP client harness in
  `internal/mcp` tests (registration + at least one CallTool assertion).
- The `mcp` command help text and generated docs reflect the current tool
  set.

## 5. Safety & correctness review (judgment; FAIL for real bugs, WARN for smells)

Review the diff for:

- `resp.Body` closed on all paths; no `defer` accumulation inside loops.
- Contexts threaded (`cmd.Context()` → client); no `context.Background()`
  in command paths; blocking operations cancellable (Ctrl-C must work).
- Errors checked, not discarded (`_ =` needs a reason); no panics in
  command paths.
- Token/secret handling: tokens never logged (including `--verbose` output),
  never written to files other than the sanctioned hosts.json path, masked
  in any human-facing display.
- Concurrency: shared state guarded; goroutines have exit paths.

## 6. Performance (WARN unless egregious)

- No per-item API calls inside loops when a filtered/batched endpoint
  exists (rate budget is 15 req/min for non-admins — treat requests as
  expensive).
- Unbounded reads (`io.ReadAll` without `LimitReader`) on network input.
- Pagination respects `--limit`; no accidental full-depletion.
- Obvious allocation waste in hot paths (string concat in loops, etc.) —
  only flag when it matters; this is a network-bound CLI.

## 7. Docs & hygiene

- CLAUDE.md conventions/decisions updated if behavior or standards changed.
- README updated for user-visible changes (new commands/flags/config).
- New flags documented in command help text; tests exist for new logic
  (unit or httptest; stub-server smoke for new commands).

## Report format

End with a compact report: one line per section (PASS/WARN/FAIL + short
reason), then a verdict: **READY TO COMMIT** or **BLOCKED** with the
ordered fix list. Offer to fix findings; re-run the failed sections after
fixing. Do not commit as part of this skill — committing stays with /jc or
the user.
