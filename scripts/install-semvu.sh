#!/usr/bin/env bash
#
# Installs semvu from a GitHub release: downloads the archive for this machine,
# checks it against the release's checksums, and puts semvu on the PATH.
#
# The semvu GitHub Action (action.yml) runs this. It also works on its own, in
# any CI system or shell with bash, curl and tar:
#
#   curl -fsSL https://raw.githubusercontent.com/tedslittlerobot/semvu/main/scripts/install-semvu.sh | bash
#
# Settings, all optional, from the environment:
#
#   SEMVU_VERSION       The version to install, as in 0.1.0 or v0.1.0, or
#                     "latest" (the default) for the latest release.
#   SEMVU_INSTALL_DIR   Where to put the semvu binary. Defaults to a directory
#                     under $RUNNER_TEMP in GitHub Actions, or ~/.local/bin.
#   GH_TOKEN          A GitHub token, used to look up the latest release
#                     without hitting the API's anonymous rate limit.
#   SEMVU_DOWNLOAD_URL  Where releases are downloaded from. Only for testing.

set -euo pipefail

repo="tedslittlerobot/semvu"
version="${SEMVU_VERSION:-latest}"
download_url="${SEMVU_DOWNLOAD_URL:-https://github.com/$repo/releases/download}"

fail() {
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::error::$1"
  fi

  echo "Error: $1" >&2
  exit 1
}

# The latest release, which never includes pre-releases.
if [ "$version" = "latest" ]; then
  api="https://api.github.com/repos/$repo/releases/latest"

  if [ -n "${GH_TOKEN:-}" ]; then
    release="$(curl -fsSL -H "Authorization: Bearer $GH_TOKEN" "$api")" || fail "semvu could not look up its latest release."
  else
    release="$(curl -fsSL "$api")" || fail "semvu could not look up its latest release."
  fi

  version="$(printf '%s\n' "$release" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$version" ] || fail "semvu has no release to install yet."
fi

version="${version#v}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "semvu has no release for $(uname -s). Releases are built for Linux and macOS." ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  armv7l | armv7) arch=armv7 ;;
  i386 | i686) arch=386 ;;
  *) fail "semvu has no release for the $(uname -m) architecture." ;;
esac

archive="semvu_${version}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -f "$tmp/$archive" "$tmp/checksums.txt" "$tmp/semvu" "$tmp/LICENSE" "$tmp/README.md"; rmdir "$tmp"' EXIT

curl -fsSL -o "$tmp/$archive" "$download_url/v$version/$archive" || fail "semvu could not download $archive for version $version."
curl -fsSL -o "$tmp/checksums.txt" "$download_url/v$version/checksums.txt" || fail "semvu could not download the checksums for version $version."

# Check the archive against its line in checksums.txt.
expected="$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)" || true
[ -n "$expected" ] || fail "checksums.txt for version $version has no entry for $archive."

if command -v sha256sum > /dev/null; then
  actual="$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)"
else
  actual="$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)"
fi

[ "$actual" = "$expected" ] || fail "$archive doesn't match its checksum, so it wasn't installed."

if [ -n "${SEMVU_INSTALL_DIR:-}" ]; then
  dir="$SEMVU_INSTALL_DIR"
elif [ -n "${RUNNER_TEMP:-}" ]; then
  dir="$RUNNER_TEMP/semvu/bin"
else
  dir="$HOME/.local/bin"
fi

mkdir -p "$dir"
tar -xzf "$tmp/$archive" -C "$tmp" semvu
install -m 0755 "$tmp/semvu" "$dir/semvu"

# In GitHub Actions, put semvu on the PATH for later steps, and say which
# version was installed.
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$dir" >> "$GITHUB_PATH"
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "version=$version" >> "$GITHUB_OUTPUT"
fi

echo "Installed semvu $version to $dir/semvu."
