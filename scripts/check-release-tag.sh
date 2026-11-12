#!/usr/bin/env bash
#
# Checks that a release tag is one semvu should publish:
#
#   1. it is a valid semvu version (semvu validate); and
#   2. it is the latest version on its own history (semvu latest), so it can't
#      go backwards. A hotfix such as v1.2.4 on a maintenance branch is fine,
#      because v1.3.0 on main isn't on its history.
#
# The release workflow runs it with semvu built from the tagged commit, so every
# release also tests semvu on itself. Run it from the repository root, with the
# tag checked out:
#
#   scripts/check-release-tag.sh v1.2.3 [path/to/semvu]

set -euo pipefail

tag="${1:?usage: scripts/check-release-tag.sh <tag> [path/to/semvu]}"
semvu="${2:-semvu}"

fail() {
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::error::$1"
  fi

  echo "Error: $1" >&2
  exit 1
}

# semvu validate explains what's wrong on stderr.
if ! "$semvu" validate "$tag"; then
  fail "$tag is not a valid semvu version, so it can't be released."
fi

latest="$("$semvu" latest --with-v)"

if [ "$latest" != "$tag" ]; then
  fail "$tag is older than $latest, which is already on its history. Tag a newer version."
fi

echo "$tag is a valid semvu version, and the latest on its history."
