# svu — Semantic Version Utilities

[![CI](https://github.com/tedslittlerobot/svm/actions/workflows/ci.yml/badge.svg)](https://github.com/tedslittlerobot/svm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tedslittlerobot/svm)](https://github.com/tedslittlerobot/svm/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/tedslittlerobot/svm.svg)](https://pkg.go.dev/github.com/tedslittlerobot/svm)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**svu is a set of small, predictable tools for working with
[semantic versions](https://semver.org).**

Its first tool is `svu modify`. You give it a version and ask for a change,
such as a new patch release or the next release candidate. It prints the
modified version and nothing else, so it fits neatly into scripts, CI jobs and
release processes.

> **Status:** early development. `svu modify` works, but details may still
> change. Expect breaking changes until v1.0.0.

## Version format

svu works with a strict subset of SemVer:

```text
[v]MAJOR.MINOR.PATCH[-STAGE-INDEX]      e.g. 1.2.3, v1.2.3, 1.2.3-alpha-001, v2.3.4-rc-987
```

Stages are `alpha`, `beta` and `rc`, always in that order. The index is three
digits, from `001` to `999`. Build metadata (`+…`) isn't supported. Other SemVer
tools sort svu versions in the same order svu does. See
[`conventions/semver.md`](conventions/semver.md) for the full rules.

## Installation

### Pre-built binaries

Every tagged release publishes binaries for:

| OS    | Architectures                                           |
|-------|---------------------------------------------------------|
| Linux | `amd64`, `arm64`, `armv7`, `386`                        |
| macOS | `amd64` (Intel), `arm64` (Apple silicon)                |

Download the archive for your platform from the
[releases page](https://github.com/tedslittlerobot/svm/releases), extract it,
and put `svu` somewhere on your `PATH`.

### Linux packages

Each release also includes native packages:

| Distribution family          | Format       | Install with                       |
|------------------------------|--------------|------------------------------------|
| Debian, Ubuntu               | `.deb`       | `sudo apt install ./svu_*.deb`     |
| Fedora, RHEL, openSUSE       | `.rpm`       | `sudo dnf install ./svu-*.rpm`     |
| Alpine                       | `.apk`       | `sudo apk add --allow-untrusted ./svu_*.apk` |
| Arch, Manjaro                | `.pkg.tar.zst` | `sudo pacman -U ./svu-*.pkg.tar.zst` |

### From source

With Go installed (see `go.mod` for the minimum version):

```sh
go install github.com/tedslittlerobot/svm/cmd/svu@latest
```

## Usage

```sh
svu help             # list commands (also svu --help, or just svu)
svu latest           # print the latest version in the git tags
svu help modify      # help for one command (also svu modify --help)
```

### svu latest

`svu latest` (or `svu l`) prints the latest version in the git tags of the
repository that contains the current directory, or `--directory`. Tags that
aren't valid versions are skipped and any `v` prefix is dropped: the same
tags, filtering and ordering that `svu modify` uses when given no version.

```sh
svu latest                     # → e.g. 1.3.0-rc-002
svu latest --release           # skip pre-releases → e.g. 1.2.0
svu l --directory ../app       # ...of the repository at ../app
```

It's an error if there is no repository, no valid tag, or, with `--release`,
no released version.

### svu modify

`svu modify` (or `svu m`) reads one version, checks it against the
[version format](#version-format), applies any [modifiers](#modifiers) and
prints it:

```sh
svu modify 1.2.3-rc-001            # from an argument                → 1.2.3-rc-001
echo v1.2.3 | svu modify -         # from standard input, with "-"    → 1.2.3
svu modify                         # from the latest valid git tag    → e.g. 0.10.0
svu modify --directory ../app      # ...of the repository at ../app
svu modify --with-v-prefix 1.2.3   # add a "v" prefix (alias --vp)   → v1.2.3
```

With no argument, svu modify reads the tags of the git repository that contains the
current directory (or `--directory`), ignores any that aren't valid versions,
and uses the latest. It's an error if there is no repository or no valid tag.

svu modify only reads standard input when the argument is `-`. It never reads it
otherwise, so it can't hang in CI jobs, git hooks or loops where standard input
is open but not meant for svu.

The `v` prefix is never printed unless you ask for it, whatever the input had.

#### Modifiers

Modifiers change the version before svu modify prints it. Use at most one version
modifier and at most one stage modifier.

| Modifier | Effect | Example |
|---|---|---|
| `--major` | Bump MAJOR, reset MINOR and PATCH to 0 | `1.2.3` → `2.0.0` |
| `--minor` | Bump MINOR, reset PATCH to 0 | `1.2.3` → `1.3.0` |
| `--patch` | Bump PATCH | `1.2.3` → `1.2.4` |
| `--alpha`, `--beta`, `--rc` | Move to a later stage at `001`, or to the next index in the same stage | `1.2.3-alpha-099` → `1.2.3-rc-001`, `1.2.3-rc-001` → `1.2.3-rc-002` |
| `--release` | Remove the stage from a pre-release | `1.2.3-rc-001` → `1.2.3` |

A version modifier on its own removes any stage: `1.2.3-rc-004 --patch` gives
`1.2.4`. Put one of each together to bump the version and start a stage at
`001`:

```sh
svu modify 1.2.3 --minor --alpha   # 1.3.0-alpha-001
svu m --patch --rc --vp            # next patch's first release candidate, from the latest git tag
```

Stages only move forward, as set out in
[`conventions/semver.md`](conventions/semver.md#stage-progression). svu
refuses to:

- release a version that is already released (`1.2.3 --release`)
- put a release back into a stage (`1.2.3 --alpha`), so start a new version
  instead, as in `--patch --alpha`
- go back to an earlier stage (`1.2.3-rc-001 --alpha`)
- go past index `999`

### Other commands

```sh
svu --version    # print svu's own version, as in 1.4.0
svu version      # print svu's version with its build information
```

Shell completion scripts are available for bash, zsh, fish and PowerShell:

```sh
svu completion --help
```

### Output

svu is built for scripts:

- **stdout** only ever holds a command's main output and a newline, with no
  colour: the version for `svu modify` and `svu latest`, or svu's own version for
  `svu --version` and `svu version`. On failure it is empty.
- **stderr** gets everything else, including errors, warnings and help, in
  colour when it is a terminal.
- Colour is used on stderr only when it is a terminal. Set
  [`NO_COLOR`](https://no-color.org) or pass `--colour=never` to turn it off.

```sh
next="$(svu modify …)" || exit 1   # $next is always a valid version, or the script stops
```

See [`conventions/output.md`](conventions/output.md) for the full rules.

## Development

You need Go (the version in `go.mod`), plus
[golangci-lint](https://golangci-lint.run) v2 and, if you want to build release
artefacts locally, [GoReleaser](https://goreleaser.com) v2.

```sh
make help       # list all targets
make build      # build ./bin/svu
make test       # run tests with the race detector, in random order
make cover      # run tests and open a coverage report
make lint       # run golangci-lint
make check      # tidy, format, lint and test: run this before pushing
make snapshot   # build every release artefact into ./dist without publishing
```

### Project layout

```text
cmd/svu/            Entry point. Wires OS streams and signals into the CLI; no logic.
internal/cli/       Cobra commands. Parses input, calls domain code, formats output.
internal/buildinfo/ Version, commit and build date of the running binary.
internal/semver/    Parsing, ordering and formatting of svu versions.
internal/git/       Reads tags by running the git command.
internal/colour/    Decides when stderr uses colour, and applies it.
```

Domain logic (parsing and modifying versions) lives in its own packages under
`internal/`, independent of the CLI, so it can be tested directly.

### Contributing

1. Open an issue or pull request against `main`.
2. Make sure `make check` passes; CI runs the same checks on Linux and macOS.
3. Use [Conventional Commits](https://www.conventionalcommits.org) for commit
   messages (`feat:`, `fix:`, `docs:` and so on). Release notes are generated
   from them.

## Releasing

Releases are fully automated. Push a semver tag and GitHub Actions runs the
test suite, then builds and publishes everything with GoReleaser:

```sh
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Tags with a pre-release suffix (for example `v0.2.0-rc.1`) are published as
GitHub pre-releases.

## License

[MIT](LICENSE)
