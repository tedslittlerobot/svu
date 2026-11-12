# svm — SemVer Modifier

[![CI](https://github.com/tedslittlerobot/svm/actions/workflows/ci.yml/badge.svg)](https://github.com/tedslittlerobot/svm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tedslittlerobot/svm)](https://github.com/tedslittlerobot/svm/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/tedslittlerobot/svm.svg)](https://pkg.go.dev/github.com/tedslittlerobot/svm)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`svm` is a small command-line tool for reading and modifying
[semantic version](https://semver.org) strings.

> **Status:** early development. The project skeleton is in place and the
> version-manipulation commands are still being written. Expect breaking
> changes until v1.0.0.

## Version format

svm works with a strict subset of SemVer:

```text
[v]MAJOR.MINOR.PATCH[-STAGE-INDEX]      e.g. 1.2.3, v1.2.3, 1.2.3-alpha-001, v2.3.4-rc-987
```

Stages are `alpha`, `beta` and `rc`, always in that order. The index is three
digits, from `001` to `999`. Build metadata (`+…`) isn't supported. Other SemVer
tools sort svm versions in the same order svm does. See
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
and put `svm` somewhere on your `PATH`.

### Linux packages

Each release also includes native packages:

| Distribution family          | Format       | Install with                       |
|------------------------------|--------------|------------------------------------|
| Debian, Ubuntu               | `.deb`       | `sudo apt install ./svm_*.deb`     |
| Fedora, RHEL, openSUSE       | `.rpm`       | `sudo dnf install ./svm-*.rpm`     |
| Alpine                       | `.apk`       | `sudo apk add --allow-untrusted ./svm_*.apk` |
| Arch, Manjaro                | `.pkg.tar.zst` | `sudo pacman -U ./svm-*.pkg.tar.zst` |

### From source

With Go installed (see `go.mod` for the minimum version):

```sh
go install github.com/tedslittlerobot/svm/cmd/svm@latest
```

## Usage

```sh
svm --help       # list commands
svm --version    # print the version
svm version      # print full build information
```

Shell completion scripts are available for bash, zsh, fish and PowerShell:

```sh
svm completion --help
```

More commands will be documented here as they land.

## Development

You need Go (the version in `go.mod`), plus
[golangci-lint](https://golangci-lint.run) v2 and, if you want to build release
artefacts locally, [GoReleaser](https://goreleaser.com) v2.

```sh
make help       # list all targets
make build      # build ./bin/svm
make test       # run tests with the race detector, in random order
make cover      # run tests and open a coverage report
make lint       # run golangci-lint
make check      # tidy, format, lint and test: run this before pushing
make snapshot   # build every release artefact into ./dist without publishing
```

### Project layout

```text
cmd/svm/            Entry point. Wires OS streams and signals into the CLI; no logic.
internal/cli/       Cobra commands. Parses input, calls domain code, formats output.
internal/buildinfo/ Version, commit and build date of the running binary.
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
