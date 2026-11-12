# svu — Semantic Version Utilities

[![CI](https://github.com/tedslittlerobot/svu/actions/workflows/ci.yml/badge.svg)](https://github.com/tedslittlerobot/svu/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tedslittlerobot/svu)](https://github.com/tedslittlerobot/svu/releases)
[![Licence: MIT](https://img.shields.io/badge/Licence-MIT-blue.svg)](LICENSE)

**svu is a set of small, predictable tools for working with
[semantic versions](https://semver.org).**

It answers two questions that come up in every release process:

- **What's the latest version?** `svu latest` reads your git tags and prints
  the newest valid version.
- **What comes next?** `svu modify` takes a version (or your latest tag) and
  bumps it, moves it to a new stage, or releases it.

Each prints a version and nothing else, so you can drop it straight into a
script or CI job:

```sh
$ svu latest
1.2.3
$ svu modify --minor --rc --with-v-prefix
v1.3.0-rc-001
```

> **Status:** early development. Expect breaking changes until v1.0.0.

## Contents

- [Installation](#installation)
- [Setup](#setup)
- [Quick start](#quick-start)
- [Version format](#version-format)
- [Commands](#commands): [which tags svu reads](#which-tags-svu-reads),
  [`svu latest`](#svu-latest), [`svu modify`](#svu-modify),
  [`svu validate`](#svu-validate), [others](#other-commands)
- [Recipes](#recipes)
- [Scripting](#scripting)
- [Development](#development)

## Installation

svu is a single binary with no dependencies. It needs
[git](https://git-scm.com) on your `PATH` only to read tags (`svu latest`, or
`svu modify` with no version).

Pick whichever way suits you. Releases are listed on the
[releases page](https://github.com/tedslittlerobot/svu/releases). In the
examples below, replace `0.1.0` with the version you want.

### Pre-built binary (Linux and macOS)

| OS    | Architectures                              |
|-------|--------------------------------------------|
| Linux | `amd64`, `arm64`, `armv7`, `386`           |
| macOS | `amd64` (Intel), `arm64` (Apple silicon)   |

Download the archive for your platform, extract `svu` and put it on your
`PATH`:

```sh
VERSION=0.1.0
OS=linux     # or darwin for macOS
ARCH=amd64   # or arm64, armv7, 386

curl -fsSLO "https://github.com/tedslittlerobot/svu/releases/download/v${VERSION}/svu_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "svu_${VERSION}_${OS}_${ARCH}.tar.gz" svu
sudo install svu /usr/local/bin/svu
```

To check the download first, fetch `checksums.txt` from the same release:

```sh
curl -fsSLO "https://github.com/tedslittlerobot/svu/releases/download/v${VERSION}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt         # Linux
shasum -a 256 --check --ignore-missing checksums.txt     # macOS
```

Each release is also signed with build provenance, which proves the archive
was built by this repository's release workflow. To check it, with the
[GitHub CLI](https://cli.github.com):

```sh
gh attestation verify "svu_${VERSION}_${OS}_${ARCH}.tar.gz" --repo tedslittlerobot/svu
```

> **macOS:** if you download the archive with a browser rather than `curl`,
> macOS may refuse to run the binary. Clear the quarantine flag with
> `xattr -d com.apple.quarantine svu`.

### Linux packages

Each release includes native packages, which install `svu` to `/usr/bin`.
Download the one for your distribution from the release, then install it:

| Distribution family    | File (amd64 shown)                   | Install with                                     |
|------------------------|--------------------------------------|--------------------------------------------------|
| Debian, Ubuntu         | `svu_0.1.0_amd64.deb`                | `sudo apt install ./svu_0.1.0_amd64.deb`         |
| Fedora, RHEL, openSUSE | `svu-0.1.0-1.x86_64.rpm`             | `sudo dnf install ./svu-0.1.0-1.x86_64.rpm`      |
| Alpine                 | `svu_0.1.0_x86_64.apk`               | `sudo apk add --allow-untrusted ./svu_0.1.0_x86_64.apk` |
| Arch, Manjaro          | `svu-0.1.0-1-x86_64.pkg.tar.zst`     | `sudo pacman -U ./svu-0.1.0-1-x86_64.pkg.tar.zst` |

For example, on Debian or Ubuntu:

```sh
curl -fsSLO "https://github.com/tedslittlerobot/svu/releases/download/v0.1.0/svu_0.1.0_amd64.deb"
sudo apt install ./svu_0.1.0_amd64.deb
```

### With Go

If you have Go installed (see `go.mod` for the minimum version):

```sh
go install github.com/tedslittlerobot/svu/cmd/svu@latest
```

This puts `svu` in `$(go env GOPATH)/bin`, which needs to be on your `PATH`.

## Setup

### Check it works

```sh
svu --version    # prints the installed version, as in 0.1.0
svu help         # lists the commands (just svu does the same)
```

### Shell completion (optional)

svu can complete its commands and flags when you press <kbd>Tab</kbd>. Add it
for your shell:

**bash** (needs the `bash-completion` package):

```sh
svu completion bash | sudo tee /etc/bash_completion.d/svu > /dev/null
```

**zsh**:

```sh
echo 'autoload -U compinit; compinit' >> ~/.zshrc        # if completion isn't on already
svu completion zsh > "${fpath[1]}/_svu"
```

**fish**:

```sh
svu completion fish > ~/.config/fish/completions/svu.fish
```

Then start a new shell. `svu completion <shell> --help` has more options,
including PowerShell.

### In CI

On **GitHub Actions**, use the svu action. It downloads the release binary,
checks it against the release's checksums, and puts `svu` on the `PATH`. It
doesn't need Go.

```yaml
- uses: actions/checkout@v5
  with:
    fetch-depth: 0   # svu needs the tags; a shallow clone has none

- uses: tedslittlerobot/svu@v0.1.0
  with:
    version: 0.1.0   # optional; the default is the latest release

- run: echo "Next version: $(svu modify --patch --vp)"
```

The action runs on Linux and macOS runners, and sets a `version` output with
the svu version it installed.

On **other CI systems**, run the same install script:

```sh
curl -fsSL https://raw.githubusercontent.com/tedslittlerobot/svu/main/scripts/install-svu.sh \
  | SVU_VERSION=0.1.0 bash   # installs to ~/.local/bin
```

Either way, svu reads tags, and many CI systems clone without them. Fetch the
full history (`fetch-depth: 0` above), or run `git fetch --tags` first.

## Quick start

In a git repository with a tag such as `v1.2.3`:

```sh
svu latest                      # 1.2.3            the latest version
svu modify --patch              # 1.2.4            the next patch release
svu modify --minor --rc         # 1.3.0-rc-001     the first release candidate of the next minor
svu modify 1.3.0-rc-001 --rc    # 1.3.0-rc-002     the next release candidate
svu modify 1.3.0-rc-002 --release  # 1.3.0         the release itself
```

Add `--with-v-prefix` (or `--vp`) to get `v1.2.4` instead of `1.2.4`, for
example to use as a tag name.

## Version format

svu works with a strict subset of SemVer:

```text
[v]MAJOR.MINOR.PATCH[-STAGE-INDEX]      e.g. 1.2.3, v1.2.3, 1.2.3-alpha-001, v2.3.4-rc-987
```

- **Stages** are `alpha`, `beta` and `rc`, always in that order. A version with
  no stage is a **release**.
- The **index** is always three digits, from `001` to `999`.
- A leading `v` is accepted on input and ignored. svu only prints one when you
  ask with `--with-v-prefix`.
- Build metadata (`+…`) isn't supported.

These rules keep svu's ordering identical to plain SemVer's, so other tools
(`sort -V`, Go modules, package registries) order svu versions the same way.
[`conventions/semver.md`](conventions/semver.md) has the full rules.

If you give svu something it can't accept, it says exactly what's wrong, and
suggests a fix where it can:

```text
$ svu modify 1.2.3-alpha-1
Error: "1.2.3-alpha-1" is not a valid version.

  1.2.3-alpha-1
              ^
  The stage index must be exactly three digits, from 001 to 999.

Hint: Did you mean "1.2.3-alpha-001"?
```

## Commands

Run `svu help <command>` (or `svu <command> --help`) for full help on any
command.

### Which tags svu reads

`svu latest`, and `svu modify` without a version, read git tags the same way:

- **Only tags on the checked-out commit's history count**, so on a
  maintenance branch svu sees that branch's versions, not newer ones from
  `main`. Add `--global` to read every tag in the repository.
- **Tags that aren't valid versions are skipped** (such as `nightly`), and any
  `v` is ignored.
- **The latest is chosen by svu's ordering**, not by tag date or name.

These flags apply to both commands:

| Flag | Meaning |
|---|---|
| `--directory path` | Read the repository containing `path` instead of the current directory |
| `--global` | Read every tag, not only those on the checked-out commit's history |
| `--tag-prefix prefix` | Only read tags starting with `prefix`, and ignore the prefix: with `--tag-prefix api/`, the tag `api/v1.2.3` is version `1.2.3` |
| `--initial version` | Use `version` if no tag matches, as in a new repository, instead of failing |

### svu latest

`svu latest` (alias `svu l`) prints the latest version in the git tags.

```sh
svu latest                     # 1.3.0-rc-002     the latest version, pre-releases included
svu latest --released          # 1.2.0            the latest release, skipping pre-releases
svu latest --with-v-prefix     # v1.3.0-rc-002
svu latest --global            # including tags on other branches
svu latest --tag-prefix api/   # the latest api/… tag, as a version
svu latest --initial 0.0.0     # 0.0.0 if there are no matching tags yet
svu l --directory ../app       # the latest version in the repository at ../app
```

| Flag | Meaning |
|---|---|
| `--released` | Skip pre-releases |
| `--at-head` | Only read tags on the checked-out commit itself. It fails if there are none, so a script can check whether a commit is tagged yet |
| `--with-v-prefix`, `--vp` | Print a leading `v` |

It also takes [the tag flags](#which-tags-svu-reads). Without `--initial`, it's
an error if there is no repository, no tag that is a valid version, or, with
`--released`, no released version.

### svu modify

`svu modify` (alias `svu m`) reads one version, applies any modifiers, and
prints the result.

#### Where the version comes from

| You run | svu reads |
|---|---|
| `svu modify 1.2.3` | The argument |
| `echo 1.2.3 \| svu modify -` | Standard input, because the argument is `-` |
| `svu modify` | The latest git tag, exactly as `svu latest` finds it, using [the tag flags](#which-tags-svu-reads) |
| `svu modify --hotfix` | The latest *released* git tag |

svu only reads standard input when you pass `-`. It never reads it otherwise,
so it can't hang in CI jobs, git hooks or loops where standard input is open
but not meant for svu.

#### Modifiers

Use at most one version modifier and at most one stage modifier.

| Modifier | Effect | Example |
|---|---|---|
| `--major` | Bump MAJOR, reset MINOR and PATCH to 0 | `1.2.3` → `2.0.0` |
| `--minor` | Bump MINOR, reset PATCH to 0 | `1.2.3` → `1.3.0` |
| `--patch` | Bump PATCH | `1.2.3` → `1.2.4` |
| `--hotfix` | Bump PATCH of the latest release, skipping pre-releases | tags `v1.2.3`, `v1.3.0-rc-001` → `1.2.4` |
| `--alpha`, `--beta`, `--rc` | Move to that stage at `001`, or to the next index if already there | `1.2.3-alpha-099` → `1.2.3-rc-001`<br>`1.2.3-rc-001` → `1.2.3-rc-002` |
| `--release` | Remove the stage from a pre-release | `1.2.3-rc-001` → `1.2.3` |

- A version modifier **on its own removes any stage**: `1.2.3-rc-004 --patch`
  gives `1.2.4`. To release `1.2.3-rc-004` as `1.2.3`, use `--release`.
- A version modifier **with a stage modifier** bumps the version and starts
  that stage at `001`: `1.2.3 --minor --alpha` gives `1.3.0-alpha-001`.
- **`--hotfix`** is for fixing what's already out while a pre-release is in
  progress. From git tags, it reads the latest *release*, so `v1.3.0-rc-001`
  doesn't get in the way. `--patch` there would give `1.3.1`. Given a version
  directly, it must be a release.
- With **no modifiers**, svu prints the version as it is. That's useful for
  checking or normalising one (`v1.2.3` → `1.2.3`).

Stages only move forward, as set out in
[`conventions/semver.md`](conventions/semver.md#stage-progression). svu
refuses to:

- release a version that is already released (`1.2.3 --release`)
- put a release back into a stage (`1.2.3 --alpha`). Start a new version
  instead, as in `--patch --alpha`.
- go back to an earlier stage (`1.2.3-rc-001 --alpha`)
- go past index `999`

Other flags:

| Flag | Meaning |
|---|---|
| `--with-v-prefix`, `--vp` | Print a leading `v` |

With no version given, it also takes [the tag flags](#which-tags-svu-reads).
They can't be used with a version, because there are no tags to read.

### svu validate

`svu validate` checks that a version is valid. It prints nothing: the exit
status is the answer, `0` if valid and `3` if not, with the reason on
standard error.

```sh
svu validate 1.2.3-rc-001                    # exit status 0
svu validate 1.2.3-alpha-1                   # exit status 3, and the reason
echo "$TAG" | svu validate -                 # read the version from standard input
svu validate "$TAG" 2>/dev/null && echo ok   # just the answer
```

### Other commands

```sh
svu help [command]        # general help, or help for one command, on stdout (also just svu)
svu <command> --help      # the same help, on stderr (also -h)
svu --version             # svu's own version, as in 0.1.0
svu version               # svu's version with its build information
svu completion <shell>    # a completion script; see Setup
svu validate <version>    # check a version, by exit status; see above
```

Every command accepts `--colour auto|always|never` (alias `--color`).

## Recipes

### A release-candidate cycle

```sh
svu latest --vp                    # v1.2.3          what's out now

git tag "$(svu m --minor --rc --vp)"      # tags v1.3.0-rc-001
git tag "$(svu m --rc --vp)"              # tags v1.3.0-rc-002
git tag "$(svu m --release --vp)"         # tags v1.3.0
```

Each `svu m` reads the tag created by the line before, so every step builds on
the last.

### Tag the next patch release in CI

```sh
# If this commit already has a version tag, as when a job is re-run, stop.
status=0
svu latest --at-head > /dev/null 2>&1 || status=$?
if [ "$status" -eq 0 ]; then echo "Already tagged."; exit 0; fi
if [ "$status" -ne 6 ]; then exit "$status"; fi   # 6: no version tag yet

next="$(svu modify --patch --with-v-prefix)" || exit 1
git tag -a "$next" -m "$next"
git push origin "$next"
```

Without the check, re-running the job would tag the same commit a second
time, with the next version after that.

### Check a version before using it

```sh
if svu validate "$INPUT" 2>/dev/null; then
  echo "valid"
else
  echo "not a valid svu version: $INPUT"
fi
```

### Hotfix a release while a release candidate is in progress

```sh
svu latest --vp                    # v1.3.0-rc-002    the rc in progress
svu latest --released --vp         # v1.2.3           what's out now
git tag "$(svu m --hotfix --vp)"   # tags v1.2.4
```

### Start versioning a new repository

```sh
git tag "$(svu m --initial 0.0.0 --minor --vp)"   # tags v0.1.0 when there are no tags yet
```

With tags present, `--initial` is ignored, so the same line works on every
run.

### Version one package in a monorepo

```sh
next="$(svu m --tag-prefix api/ --minor --vp)"   # api/v1.4.0 → v1.5.0
git tag "api/$next"
```

### Work on a maintenance branch

```sh
git switch support/1.x
svu latest                         # 1.2.5    this branch's latest, not main's 2.0.0
git tag "$(svu m --patch --vp)"    # tags v1.2.6
```

### Version a project in another directory

```sh
svu latest --directory ../api --released
```

## Scripting

svu is built to be captured with `$(…)`:

- **stdout** holds the command's main output and nothing else: one version and
  a newline for `svu latest` and `svu modify`. It never contains colour, and
  it is **empty whenever svu fails**. (`svu help`, and `svu` on its own,
  write help there too, so `svu help | less` works.)
- **stderr** gets everything else: errors, hints, and help from `--help`.
- **Exit status** tells you what went wrong without reading stderr:

| Status | Meaning |
|---|---|
| `0` | Success |
| `1` | Internal failure, such as a bug; please report it |
| `2` | Usage error: unknown command or flag, bad flag value, conflicting flags |
| `3` | Invalid version |
| `4` | The version can't be modified as asked, such as going back a stage |
| `5` | No git repository where svu looked |
| `6` | No tag that is a version of the kind asked for |
| `7` | git isn't installed, or failed |

So this either gives a valid version, or stops the script with an empty
variable:

```sh
next="$(svu modify --patch)" || exit 1
```

And this starts from scratch only when there really are no matching tags:

```sh
latest="$(svu latest)"; status=$?
if [ "$status" -eq 6 ]; then latest=0.0.0; elif [ "$status" -ne 0 ]; then exit "$status"; fi
```

(`svu latest --initial 0.0.0` does the same in one step.)

Colour is used only when the stream is a terminal. Set
[`NO_COLOR`](https://no-color.org) or pass `--colour=never` to turn it off, or
`--colour=always` (or `FORCE_COLOR`) to force it on.
[`conventions/output.md`](conventions/output.md) has the full rules.

## Development

You need:

- **Go**, at the version in `.go-version`. svu always uses the latest Go
  release.
- [golangci-lint](https://golangci-lint.run) v2, built with that Go version.
- [GoReleaser](https://goreleaser.com) v2, if you want to build release
  artefacts locally.

### Set up Go with goenv

We recommend [goenv](https://github.com/go-nv/goenv) for managing Go. It reads
`.go-version` and switches to that version whenever you're in this repository,
so you're always on the version svu builds with, whatever else you have
installed.

```sh
goenv install --skip-existing   # install the Go version in .go-version
go version                      # check: it should match .go-version
```

When svu moves to a new Go release, pull and run the same `goenv install`
again.

Without goenv, install the version in `.go-version` some other way. Go's own
toolchain switching (`GOTOOLCHAIN=auto`, the default) will also fetch it for
you, as long as your installed Go is 1.21 or newer.

### Make targets

```sh
make help              # list all targets
make build             # build ./bin/svu
make test              # run tests with the race detector, in random order
make cover-check       # run tests, and fail if coverage is below 95% (CI runs this)
make fuzz              # fuzz the version parser for 30 seconds (CI runs this)
make cover             # run tests and open a coverage report
make lint              # run golangci-lint
make vuln              # check for known vulnerabilities that svu's code can reach
make go-latest         # move go.mod and .go-version to the latest Go release
make go-version-check  # check .go-version matches go.mod
make check             # tidy, format, lint and test: run this before pushing
make snapshot          # build every release artefact into ./dist without publishing
```

### Project layout

```text
cmd/svu/            Entry point. Wires OS streams and signals into the CLI; no logic.
internal/cli/       Cobra commands. Parses input, calls domain code, formats output.
internal/buildinfo/ Version, commit and build date of the running binary.
internal/semver/    Parsing, ordering, formatting and modifying svu versions.
internal/git/       Reads tags by running the git command.
internal/colour/    Decides when stderr uses colour, and applies it.
```

Domain logic lives in its own packages under `internal/`, independent of the
CLI, so it can be tested directly.

### Contributing

1. Open an issue or pull request against `main`.
2. Make sure `make check` passes. CI runs the same checks on Linux and macOS,
   plus a coverage minimum of 95%, fuzzing, a vulnerability check and a
   release build.
3. To report a security problem, follow [`SECURITY.md`](SECURITY.md) instead
   of opening an issue.
4. Use [Conventional Commits](https://www.conventionalcommits.org) for commit
   messages (`feat:`, `fix:`, `docs:` and so on). Release notes are generated
   from them.

## Releasing

Releases are fully automated. Push a tag and GitHub Actions runs the test
suite, then builds and publishes everything with GoReleaser. svu can work out
the tag:

```sh
next="$(svu modify --minor --with-v-prefix)"   # for example v0.2.0
git tag -a "$next" -m "$next"
git push origin "$next"
```

Tags with a stage (for example `v0.2.0-rc-001`) are published as GitHub
pre-releases.

Before anything is published, the release workflow builds svu from the tagged
commit and uses it to check the tag
([`scripts/check-release-tag.sh`](scripts/check-release-tag.sh)). The tag
must be a valid svu version, and the latest version on its own history. So
`v0.2.0-rc.1` is refused, as is `v0.1.5` after `v0.2.0` on `main`, but a
hotfix such as `v0.1.5` on a maintenance branch is fine.

Each release is published with signed build provenance for every archive and
package (see [`SECURITY.md`](SECURITY.md#checking-a-download)).

## Licence

[MIT](LICENSE)
