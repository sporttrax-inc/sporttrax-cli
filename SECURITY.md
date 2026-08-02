# Security policy

## Reporting a vulnerability

Please report security issues privately. Do **not** open a public GitHub
issue, and do not include credentials or tokens in your report.

- Email: **security@sporttrax.com**
- Or use GitHub's [private vulnerability reporting][pvr] on this
  repository (Security → Report a vulnerability).

[pvr]: https://docs.github.com/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability

Include what you did, what you expected, and what happened — a minimal
reproduction, the `sporttrax --version` output, and your OS/architecture
are usually enough. We aim to acknowledge reports within a few business
days.

## Scope

This policy covers the `sporttrax` CLI in this repository: the binary, its
handling of credentials and API responses, the release artifacts, and the
MCP server it exposes.

Vulnerabilities in the SportTrax API or web application itself are **out
of scope here** — report those to the same address, noting that they
concern the server rather than the CLI.

## What matters most in this CLI

If you are looking for somewhere to start, these are the areas where a bug
would hurt most:

- **Token handling.** Tokens are stored in the OS keychain, falling back
  to `hosts.json` (0600) in the config directory. A token must only ever
  be sent to a host the user vouched for — a stock environment, one
  declared in `config.yaml`, or localhost — and must never be written to
  logs, including `--verbose` output.
- **Transport.** API URLs must be `https`; plain `http` is accepted only
  for localhost. Disabling certificate verification is refused for the
  stock environments.
- **Terminal output.** Server-supplied text (meet, athlete, team, and
  venue names are user-submitted) is sanitized of escape sequences and
  control characters before it reaches a terminal. `--json` and the MCP
  tools are deliberately verbatim, since nothing interprets them as a
  terminal.

## Supported versions

Fixes land on the latest release. There are no long-term support
branches — please upgrade before reporting.
