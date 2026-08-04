#!/bin/sh
# Assert that every path the generated Homebrew cask symlinks actually exists
# inside the release archives it points at.
#
# Declaring a completion under homebrew_casks.completions only writes the
# *_completion stanza into the cask — it does not package the file. When the
# two disagree, `brew install` links the binary, fails on the first missing
# symlink source, and rolls the whole install back, so the breakage surfaces
# only after the artifacts are public.
#
# Expects `goreleaser release --snapshot` to have populated dist/ already
# (see the package-check target in the Makefile).
set -eu

cask=dist/homebrew/Casks/sporttrax.rb

# The cask is the source of truth: read the paths back out of it rather than
# hardcoding a list here, so this cannot drift from what the cask links.
sources=$(sed -nE 's/^[[:space:]]*(binary|[a-z]+_completion)[[:space:]]+"([^"]+)".*/\2/p' "$cask")
[ -n "$sources" ] || { echo "no symlink sources found in $cask" >&2; exit 1; }

# shellcheck disable=SC2086 # deliberate split: one log line instead of four
echo "cask symlinks:" $sources

# Tarballs only: the cask covers macOS and Linux, and the Windows zip carries
# sporttrax.exe rather than the binary name the cask links.
for archive in dist/*.tar.gz; do
  listing=$(tar tzf "$archive")
  # shellcheck disable=SC2086 # deliberate split: iterate the newline-separated paths
  for source in $sources; do
    printf '%s\n' "$listing" | grep -qxF "$source" || {
      echo "$archive is missing $source, which the cask symlinks" >&2
      exit 1
    }
  done
done

echo "all cask symlink sources present in every tarball"
