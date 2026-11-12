#!/usr/bin/env bash
#
# Checks that a release tag is one svu should publish:
#
#   1. it is a valid svu version (svu validate); and
#   2. it is the latest version on its own history (svu latest), so it can't
#      go backwards. A hotfix such as v1.2.4 on a maintenance branch is fine,
#      because v1.3.0 on main isn't on its history.
#
# The release workflow runs it with svu built from the tagged commit, so every
# release also tests svu on itself. Run it from the repository root, with the
# tag checked out:
#
#   scripts/check-release-tag.sh v1.2.3 [path/to/svu]

set -euo pipefail

tag="${1:?usage: scripts/check-release-tag.sh <tag> [path/to/svu]}"
svu="${2:-svu}"

fail() {
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::error::$1"
  fi

  echo "Error: $1" >&2
  exit 1
}

# svu validate explains what's wrong on stderr.
if ! "$svu" validate "$tag"; then
  fail "$tag is not a valid svu version, so it can't be released."
fi

latest="$("$svu" latest --with-v)"

if [ "$latest" != "$tag" ]; then
  fail "$tag is older than $latest, which is already on its history. Tag a newer version."
fi

echo "$tag is a valid svu version, and the latest on its history."
