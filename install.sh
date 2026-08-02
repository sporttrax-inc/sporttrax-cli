#!/bin/sh
#
# Install the sporttrax CLI on macOS or Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/sporttrax-inc/sporttrax-cli/main/install.sh | sh
#
# Every download is checksum-verified against the checksums.txt published
# with the release. Piping a script to a shell means trusting whatever it
# fetches, so it should at least refuse to install bytes it cannot vouch
# for.
#
# Environment:
#   SPORTTRAX_VERSION      version to install (default: latest release)
#   SPORTTRAX_INSTALL_DIR  where to put the binary (default: see below)
#
# Windows is not covered — use the .zip from the releases page.

set -eu

REPO="sporttrax-inc/sporttrax-cli"
BINARY="sporttrax"

die() {
	echo "install: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"
}

need uname
need tar
need curl

# --- platform -------------------------------------------------------------

os=$(uname -s)
case "$os" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*)
	die "unsupported operating system: $os
Windows users: download the .zip from https://github.com/$REPO/releases/latest"
	;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "unsupported architecture: $arch (builds exist for amd64 and arm64)" ;;
esac

# --- version --------------------------------------------------------------

version="${SPORTTRAX_VERSION:-}"
if [ -z "$version" ]; then
	# Read the tag off the /releases/latest redirect rather than the API,
	# which is rate limited to 60 requests an hour per address — shared
	# NATs at a school would hit that.
	latest_url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
		"https://github.com/$REPO/releases/latest") ||
		die "could not reach GitHub to determine the latest version"
	version=${latest_url##*/}
fi
version=${version#v} # tags are vX.Y.Z; artifacts are X.Y.Z

[ -n "$version" ] || die "could not determine a version to install"

# --- install location -----------------------------------------------------

if [ -n "${SPORTTRAX_INSTALL_DIR:-}" ]; then
	install_dir=$SPORTTRAX_INSTALL_DIR
elif [ -w /usr/local/bin ] 2>/dev/null; then
	install_dir=/usr/local/bin
else
	# Falling back rather than reaching for sudo: a piped-in script should
	# not quietly ask for root.
	install_dir=$HOME/.local/bin
fi

# --- download and verify --------------------------------------------------

archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/v${version}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Downloading $BINARY $version ($os/$arch)..."
curl -fsSL -o "$tmp/$archive" "$base/$archive" ||
	die "could not download $archive
Check that version $version exists: https://github.com/$REPO/releases"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" ||
	die "could not download checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
	sha_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
	sha_cmd="shasum -a 256"
else
	die "need sha256sum or shasum to verify the download"
fi

echo "Verifying checksum..."
expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$expected" ] || die "$archive is not listed in checksums.txt"
actual=$(cd "$tmp" && $sha_cmd "$archive" | cut -d' ' -f1)
[ "$expected" = "$actual" ] || die "checksum mismatch for $archive
  expected $expected
  actual   $actual
Do not use this download."

# --- install --------------------------------------------------------------

tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY" || die "could not extract $BINARY"
chmod +x "$tmp/$BINARY"

mkdir -p "$install_dir" || die "could not create $install_dir"
mv "$tmp/$BINARY" "$install_dir/$BINARY" ||
	die "could not write to $install_dir
Set SPORTTRAX_INSTALL_DIR to somewhere you can write, or re-run with sudo."

echo "Installed $BINARY $version to $install_dir/$BINARY"

case ":$PATH:" in
*":$install_dir:"*) ;;
*)
	echo
	echo "$install_dir is not on your PATH. Add it:"
	echo "  export PATH=\"$install_dir:\$PATH\""
	;;
esac

echo
echo "Next: $BINARY auth login"
