# CLAUDE.md

Guidance for working in this repository.

## Project

`svm` (SemVer Modifier) is a Go command-line tool for reading and modifying
semantic version strings. It is open source under the MIT licence and hosted at
https://github.com/tedslittlerobot/svm.

- Module path: `github.com/tedslittlerobot/svm`
- Go version: the one in `go.mod`
- CLI framework: [Cobra](https://github.com/spf13/cobra)
- Spec: [`conventions/semver.md`](conventions/semver.md) defines the version
  format svm accepts: a strict subset of
  [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html) with fixed stages
  (`alpha` < `beta` < `rc` < `ga`), three-digit indexes, and its own
  precedence rules. It is the source of truth for parsing, validation,
  precedence and stage progression. Where it is silent, SemVer 2.0.0 applies.
  Where they differ (notably `rc` vs `ga` ordering), the conventions file wins.
  Cite the relevant section in comments, and turn its valid/invalid example
  tables into test cases.
- Changing `conventions/semver.md` changes behaviour. Update the doc and the
  code together, in the same commit.

## Commands

```sh
make check      # tidy, format, lint, test: must pass before any commit
make test       # go test -race -shuffle=on ./...
make lint       # golangci-lint run (config: .golangci.yml)
make fmt        # golangci-lint fmt (gofmt + goimports)
make build      # ./bin/svm with version info linked in
make snapshot   # goreleaser release --snapshot --clean (validates release config)
go test ./internal/cli -run TestVersionCommand   # run a single test
```

## Layout

```text
cmd/svm/main.go       Process boundary only: os.Args, os streams, signals, os.Exit.
internal/cli/         One file per command (<name>.go + <name>_test.go).
                      cli.go holds Execute, the root command and Streams.
internal/buildinfo/   Version metadata from ldflags, falling back to debug.ReadBuildInfo.
```

New domain logic goes in its own focused package under `internal/` (for
example, `internal/semver` for parsing and manipulation), never in `internal/cli`.
Packages are named for what they provide, not `util`, `common` or `helpers`.
Keep everything under `internal/` unless there is a deliberate decision to
publish a public Go API.

## Architecture rules

- **`main` stays trivial.** It builds `cli.Streams` from `os.Stdin`/`os.Stdout`/
  `os.Stderr`, calls `cli.Execute`, and exits. Nothing else calls `os.Exit`.
- **Commands are thin.** A command parses flags and arguments, calls domain
  code, and formats the result. Domain packages do not import Cobra, do not
  read `os.Args` or the environment, and do no I/O except through
  parameters (`io.Reader`/`io.Writer`).
- **No global I/O.** Commands write with `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`
  and read with `cmd.InOrStdin()`, never `os.Stdout`, `fmt.Println` or `log`.
- **No global mutable state.** Each command is built by an unexported
  `newXCommand(...)` constructor and registered in `newRootCommand`. Pass
  dependencies in as arguments; avoid `init()` and package-level command vars.
- **Errors.** Return errors; don't print and continue. Wrap with context using
  `fmt.Errorf("doing x: %w", err)`. Error strings are lower case with no
  trailing punctuation. `execute` prints the final error once as `svm: <err>`
  and maps it to an exit code; exit codes are defined as constants in
  `internal/cli/cli.go`, so add new ones there.
- **Output is a contract.** stdout carries only the result, so `svm` composes
  in shell pipelines; diagnostics go to stderr. Changing output format is a
  breaking change.

## Testing

Tests are expected for every change; aim for full coverage of domain packages
and of every command's success and failure paths.

- Standard library `testing` only, no assertion libraries. Use
  `github.com/google/go-cmp/cmp` if a struct diff is genuinely needed.
- Table-driven tests with descriptive `name` fields phrased as behaviour
  ("rejects positional arguments"), looped with `t.Run`.
- Call `t.Parallel()` in every test and subtest. Tests must pass under
  `-race -shuffle=on`.
- Use `t.Context()`, `t.TempDir()` and `t.Setenv()` rather than hand-rolled
  equivalents. Mark helpers with `t.Helper()`.
- Failure messages follow `got X, want Y`: `t.Errorf("Bump() = %q, want %q", got, want)`.
- Tests for unexported behaviour live in the same package; black-box tests of a
  package's public API use `package foo_test`.
- **Command tests** run the whole CLI in-process through the `run(t, args...)`
  helper in `internal/cli/cli_test.go`, which uses fixed build info and
  captures exit code, stdout and stderr. Assert all three.
- Domain packages should get fuzz tests (`FuzzXxx`) for parsers, and
  `Example` functions where they document usage well.

## Style

- Idiomatic Go per Effective Go and the Go Code Review Comments. `golangci-lint`
  enforces most of this; fix the cause rather than adding `//nolint`, and if a
  `//nolint` is unavoidable it must name the linter and give a reason.
- Every package has a package comment; every exported identifier has a doc
  comment starting with its name.
- Small functions, early returns, no naked returns, no `else` after `return`.
- Accept interfaces, return concrete types. Define interfaces where they are
  consumed, and only when there is more than one implementation or a test needs it.
- Add dependencies only when they clearly pay for themselves; prefer the
  standard library.

## Commits and releases

- Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`,
  `chore:`, `ci:`). The GitHub release changelog is generated from them and
  hides `docs:`, `test:`, `chore:` and `ci:`.
- CI (`.github/workflows/ci.yml`) runs on pull requests and pushes to `main`:
  tests on Linux and macOS, golangci-lint, and a GoReleaser snapshot build.
- Releases (`.github/workflows/release.yml`) run on `v*.*.*` tags: CI first,
  then GoReleaser publishes Linux (amd64, arm64, armv7, 386) and macOS (amd64,
  arm64) archives plus deb, rpm, apk and Arch packages to GitHub Releases.
- When changing `.goreleaser.yaml`, run `make snapshot` locally before pushing.
- Keep README.md in step with user-facing changes (commands, flags, install).
