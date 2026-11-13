# CLAUDE.md

Guidance for working in this repository.

## Goal

**The goal of this project is to make it easy to work with semantic versions.**
For now, that means modifying a version (`semvu modify`) and finding the latest
one (`semvu latest`).

Judge every feature, command, flag and design decision against this goal. Prefer
whatever makes working with a version simpler, more predictable and harder to get
wrong. Push back on anything that doesn't serve it.

## Project

`semvu` (Semantic Version Utilities) is a Go command-line tool for working with
semantic versions. Each job is a subcommand: `semvu modify` (alias `semvu m`)
and `semvu latest` (alias `semvu l`). It is open source under the MIT licence
and hosted at https://github.com/tedslittlerobot/semvu. The name is chosen to
be free on Homebrew (homebrew-core's `svu` is a different tool), npm, crates.io
and PyPI; check those before renaming anything user-facing.

- Module path: `github.com/tedslittlerobot/semvu`
- Go version: **always the latest Go release.** The `go` line in `go.mod` is
  the source of truth, and CI and release builds read it. `.go-version` holds
  the same version for [goenv](https://github.com/go-nv/goenv), the
  recommended way to manage Go for this project. `make check` fails if the
  two differ. When a new Go version is out, update in one commit:
  1. `make go-latest` (runs `go get go@latest` and `go mod tidy`, and
     rewrites `.go-version`);
  2. `goenv install --skip-existing`, which installs the version named in
     `.go-version`;
  3. bump golangci-lint in `.github/workflows/ci.yml` to a release built with
     that Go (`golangci-lint version` shows it), since older builds refuse
     newer Go;
  4. run `make check`, `make vuln` and `make snapshot`.
  The `Go version` workflow fails when `go.mod` falls behind, including on a
  weekly schedule.
- CLI framework: [Cobra](https://github.com/spf13/cobra)
- Spec: [`conventions/semver.md`](conventions/semver.md) defines the version
  format semvu accepts: a strict subset of
  [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html): fixed stages
  (`alpha` < `beta` < `rc`), three-digit indexes, and no build metadata.
  It is the source of truth for parsing, validation, precedence and stage
  progression. Where it is silent, SemVer 2.0.0 applies.
  Cite the relevant section in comments, and turn its valid/invalid example
  tables into test cases.
- Invariant: for semvu versions, semvu's ordering must match plain SemVer
  ordering. Stage names stay alphabetical and indexes stay fixed-width. Add a
  test that checks this.
- Changing `conventions/semver.md` changes behaviour. Update the doc and the
  code together, in the same commit.

## Commands

```sh
make check      # tidy, format, lint, test: must pass before any commit
make test       # go test -race -shuffle=on ./...
make cover-check  # tests with coverage; fails below 95% total (CI runs this)
make fuzz       # fuzz the version and commit parsers for 30s each (CI runs this)
make lint       # golangci-lint run (config: .golangci.yml)
make fmt        # golangci-lint fmt (gofmt + goimports)
make build      # ./bin/semvu with version info linked in
make go-latest  # move go.mod and .go-version to the latest Go release (see Project)
make go-version-check  # .go-version matches go.mod (part of make check)
make vuln       # govulncheck: known vulnerabilities semvu's code can reach (CI runs it too)
make snapshot   # goreleaser release --snapshot --clean (validates release config)
make archive-check  # build from git archive, as Homebrew does, and check the commit and date (CI runs this)
go test ./internal/cli -run TestVersionCommand   # run a single test
```

## Layout

```text
cmd/semvu/main.go     Process boundary only: os.Args, os streams, signals, os.Exit.
internal/cli/         One file per command (<name>.go + <name>_test.go).
                      cli.go holds Execute, the root command and Streams.
                      modify.go is semvu modify; modifiers.go and input.go support it.
                      latest.go is semvu latest. tags.go reads git tags for both.
                      validate.go is semvu validate. auto.go is semvu modify --auto,
                      apply.go is semvu modify --apply, push.go is --push, and
                      nogit.go is --no-git.
internal/buildinfo/   Version metadata from ldflags, falling back to debug.ReadBuildInfo.
internal/semver/      Parsing, ordering, formatting and modifying versions.
internal/commits/     Reads Conventional Commits messages and works out the bump.
internal/git/         Reads tags and commit messages by running the git command.
internal/colour/      Decides when stderr uses colour, and applies it.
action.yml            The GitHub Action that installs semvu (runs scripts/install-semvu.sh).
scripts/              Shell scripts for installing semvu and checking release tags.
                      They must run under bash 3.2 (macOS) as well as bash 5.
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
- **Output follows [`conventions/output.md`](conventions/output.md).** stdout
  is only the resulting version and a newline, with no colour, and it's empty
  on failure. Everything else goes to stderr, including help, errors and
  warnings. User-facing messages are full sentences in British English, and
  stderr may use colour as described there. Changing output is a breaking
  change.
- **Errors.** Return errors; don't print and continue. Inside the code, Go
  error values follow normal Go style: lower case, no trailing punctuation,
  wrapped with `fmt.Errorf("doing x: %w", err)`. Domain packages return typed
  errors that carry the input, position and rule that was broken. Only
  `internal/cli` turns errors into user-facing messages
  (`Error:` / detail / `Hint:`), and a raw Go error string never reaches the
  user. Exit codes are constants in `internal/cli/cli.go`, one per kind of
  failure, listed in `conventions/output.md`. Every `userError` sets one; add a
  new constant (and document it) only for a new kind of failure.

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
- **British English everywhere.** This covers identifiers (types, functions,
  variables, constants, packages, files), comments, string literals, test
  names, docs and commit messages. Write `colour`, `behaviour`, `initialise`,
  `normalise`, `licence` (noun) and `cancelled`.
  - The only exception is a name semvu doesn't control: a standard-library or
    third-party identifier (`context.Canceled`, `color.New`), a flag or
    environment variable other tools define (`NO_COLOR`), or an alias kept
    for compatibility (`--color`). Use it exactly as written there, and keep
    semvu's own wrappers and names British (`colourEnabled`, not
    `colorEnabled`).
  - `misspell` (UK locale) catches common American spellings in comments
    and strings, but it doesn't check identifiers and its word list is
    incomplete. Check identifiers by eye in review. If a string literal
    must use an American spelling, add
    `//nolint:misspell // <why the spelling is required>`.

## Commits and releases

- Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`,
  `chore:`, `ci:`). The GitHub release changelog is generated from them and
  hides `docs:`, `test:`, `chore:` and `ci:`.
- CI (`.github/workflows/ci.yml`) runs on pull requests and pushes to `main`:
  tests on Linux and macOS, golangci-lint, and a GoReleaser snapshot build.
- Releases (`.github/workflows/release.yml`) run on `v*.*.*` tags: CI first,
  then GoReleaser publishes Linux (amd64, arm64, armv7, 386) and macOS (amd64,
  arm64) archives plus deb, rpm, apk and Arch packages to GitHub Releases.
- Homebrew: the formula lives in
  [tedslittlerobot/homebrew-tap](https://github.com/tedslittlerobot/homebrew-tap)
  and builds from each release's source tarball. That repo's own workflows
  bump it after a release (pre-releases are skipped) and build bottles, so
  nothing here pushes to it. It hard-codes `./cmd/semvu` and the
  `internal/buildinfo.version` ldflag, so moving either means updating
  `Formula/semvu.rb` there too. A tarball has no git metadata, so the commit
  and date come from `internal/buildinfo/archive.go`, which git archive fills
  in (`.gitattributes` marks it `export-subst`). Keep its placeholders and
  that attribute; `make archive-check` (CI runs it) fails if either breaks.
- When changing `.goreleaser.yaml`, run `make snapshot` locally before pushing.
- Keep README.md in step with user-facing changes (commands, flags, install).
