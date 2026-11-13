# semvu — Semantic Version Utilities

[![CI](https://github.com/tedslittlerobot/semvu/actions/workflows/ci.yml/badge.svg)](https://github.com/tedslittlerobot/semvu/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tedslittlerobot/semvu)](https://github.com/tedslittlerobot/semvu/releases)
[![Licence: MIT](https://img.shields.io/badge/Licence-MIT-blue.svg)](LICENSE)

**semvu is a set of small, predictable tools for working with
[semantic versions](https://semver.org).**

It answers two questions that come up in every release process:

- **What's the latest version?** `semvu latest` reads your git tags and prints
  the newest valid version.
- **What comes next?** `semvu modify` takes a version (or your latest tag) and
  bumps it, moves it to a new stage, or releases it.

Each prints a version and nothing else, so you can drop it straight into a
script or CI job:

```sh
$ semvu latest
v1.2.3
$ semvu modify --minor --rc
v1.3.0-rc-001
```

> **Status:** early development. Expect breaking changes until v1.0.0.

## Contents

- [Installation](#installation)
- [Setup](#setup)
- [Quick start](#quick-start)
- [Version format](#version-format)
- [Commands](#commands): [which tags semvu reads](#which-tags-semvu-reads),
  [`semvu latest`](#semvu-latest), [`semvu modify`](#semvu-modify),
  [`semvu validate`](#semvu-validate), [others](#other-commands)
- [Recipes](#recipes)
- [Scripting](#scripting)
- [Development](#development)

## Installation

semvu is a single binary with no dependencies. It needs
[git](https://git-scm.com) on your `PATH` only to read tags (`semvu latest`, or
`semvu modify` with no version).

Pick whichever way suits you. Releases are listed on the
[releases page](https://github.com/tedslittlerobot/semvu/releases). In the
examples below, replace `0.1.0` with the version you want.

### With Homebrew (macOS and Linux)

```sh
brew install tedslittlerobot/tap/semvu
```

This also installs shell completions for bash, zsh and fish, so you can skip
[Shell completion](#shell-completion-optional). Upgrade with
`brew upgrade semvu`. The formula lives in
[tedslittlerobot/homebrew-tap](https://github.com/tedslittlerobot/homebrew-tap)
and picks up each new release within a day. Release candidates aren't
published there.

### Pre-built binary (Linux and macOS)

| OS    | Architectures                              |
|-------|--------------------------------------------|
| Linux | `amd64`, `arm64`, `armv7`, `386`           |
| macOS | `amd64` (Intel), `arm64` (Apple silicon)   |

Download the archive for your platform, extract `semvu` and put it on your
`PATH`:

```sh
VERSION=0.1.0
OS=linux     # or darwin for macOS
ARCH=amd64   # or arm64, armv7, 386

curl -fsSLO "https://github.com/tedslittlerobot/semvu/releases/download/v${VERSION}/semvu_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "semvu_${VERSION}_${OS}_${ARCH}.tar.gz" semvu
sudo install semvu /usr/local/bin/semvu
```

To check the download first, fetch `checksums.txt` from the same release:

```sh
curl -fsSLO "https://github.com/tedslittlerobot/semvu/releases/download/v${VERSION}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt         # Linux
shasum -a 256 --check --ignore-missing checksums.txt     # macOS
```

Each release is also signed with build provenance, which proves the archive
was built by this repository's release workflow. To check it, with the
[GitHub CLI](https://cli.github.com):

```sh
gh attestation verify "semvu_${VERSION}_${OS}_${ARCH}.tar.gz" --repo tedslittlerobot/semvu
```

> **macOS:** if you download the archive with a browser rather than `curl`,
> macOS may refuse to run the binary. Clear the quarantine flag with
> `xattr -d com.apple.quarantine semvu`.

### Linux packages

Each release includes native packages, which install `semvu` to `/usr/bin`.
Download the one for your distribution from the release, then install it:

| Distribution family    | File (amd64 shown)                 | Install with                                              |
|------------------------|------------------------------------|-----------------------------------------------------------|
| Debian, Ubuntu         | `semvu_0.1.0_amd64.deb`            | `sudo apt install ./semvu_0.1.0_amd64.deb`                |
| Fedora, RHEL, openSUSE | `semvu-0.1.0-1.x86_64.rpm`         | `sudo dnf install ./semvu-0.1.0-1.x86_64.rpm`             |
| Alpine                 | `semvu_0.1.0_x86_64.apk`           | `sudo apk add --allow-untrusted ./semvu_0.1.0_x86_64.apk` |
| Arch, Manjaro          | `semvu-0.1.0-1-x86_64.pkg.tar.zst` | `sudo pacman -U ./semvu-0.1.0-1-x86_64.pkg.tar.zst`       |

For example, on Debian or Ubuntu:

```sh
curl -fsSLO "https://github.com/tedslittlerobot/semvu/releases/download/v0.1.0/semvu_0.1.0_amd64.deb"
sudo apt install ./semvu_0.1.0_amd64.deb
```

### With Go

If you have Go installed (see `go.mod` for the minimum version):

```sh
go install github.com/tedslittlerobot/semvu/cmd/semvu@latest
```

This puts `semvu` in `$(go env GOPATH)/bin`, which needs to be on your `PATH`.

## Setup

### Check it works

```sh
semvu --version    # prints the installed version, as in 0.1.0
semvu help         # lists the commands (just semvu does the same)
```

### Shell completion (optional)

semvu can complete its commands and flags when you press <kbd>Tab</kbd>. Add it
for your shell:

**bash** (needs the `bash-completion` package):

```sh
semvu completion bash | sudo tee /etc/bash_completion.d/semvu > /dev/null
```

**zsh**:

```sh
echo 'autoload -U compinit; compinit' >> ~/.zshrc        # if completion isn't on already
semvu completion zsh > "${fpath[1]}/_semvu"
```

**fish**:

```sh
semvu completion fish > ~/.config/fish/completions/semvu.fish
```

Then start a new shell. `semvu completion <shell> --help` has more options,
including PowerShell.

### In CI

On **GitHub Actions**, use the semvu action. It downloads the release binary,
checks it against the release's checksums, and puts `semvu` on the `PATH`. It
doesn't need Go.

```yaml
- uses: actions/checkout@v5
  with:
    fetch-depth: 0   # semvu needs the tags; a shallow clone has none

- uses: tedslittlerobot/semvu@v0.1.0
  with:
    version: 0.1.0   # optional; the default is the latest release

- run: echo "Next version: $(semvu modify --patch)"
```

The action runs on Linux and macOS runners, and sets a `version` output with
the semvu version it installed.

On **other CI systems**, run the same install script:

```sh
curl -fsSL https://raw.githubusercontent.com/tedslittlerobot/semvu/main/scripts/install-semvu.sh \
  | SEMVU_VERSION=0.1.0 bash   # installs to ~/.local/bin
```

Either way, semvu reads tags, and many CI systems clone without them. Fetch the
full history (`fetch-depth: 0` above), or run `git fetch --tags` first.

## Quick start

In a git repository with a tag such as `v1.2.3`:

```sh
semvu latest                      # v1.2.3           the latest version
semvu modify --patch              # v1.2.4           the next patch release
semvu modify --minor --rc         # v1.3.0-rc-001    the first release candidate of the next minor
semvu modify 1.3.0-rc-001 --rc    # 1.3.0-rc-002     the next release candidate
semvu modify 1.3.0-rc-002 --release  # 1.3.0         the release itself
```

A version keeps its `v`, or lack of one: from the tag `v1.2.3` you get
`v1.2.4`, and from the argument `1.3.0-rc-001` you get `1.3.0-rc-002`. Add
`--with-v` or `--without-v` to choose.

## Version format

semvu works with a strict subset of SemVer:

```text
[v]MAJOR.MINOR.PATCH[-STAGE-INDEX]      e.g. 1.2.3, v1.2.3, 1.2.3-alpha-001, v2.3.4-rc-987
```

- **Stages** are `alpha`, `beta` and `rc`, always in that order. A version with
  no stage is a **release**.
- The **index** is always three digits, from `001` to `999`.
- A leading `v` is accepted on input, and ignored when comparing versions. semvu
  prints a version with a `v` if it read it with one, unless you pass
  `--with-v` or `--without-v` (see [the `v`](#the-v)).
- Build metadata (`+…`) isn't supported.

These rules keep semvu's ordering identical to plain SemVer's, so other tools
(`sort -V`, Go modules, package registries) order semvu versions the same way.
[`conventions/semver.md`](conventions/semver.md) has the full rules.

If you give semvu something it can't accept, it says exactly what's wrong, and
suggests a fix where it can:

```text
$ semvu modify 1.2.3-alpha-1
Error: "1.2.3-alpha-1" is not a valid version.

  1.2.3-alpha-1
              ^
  The stage index must be exactly three digits, from 001 to 999.

Hint: Did you mean "1.2.3-alpha-001"?
```

## Commands

Run `semvu help <command>` (or `semvu <command> --help`) for full help on any
command.

### Which tags semvu reads

`semvu latest`, and `semvu modify` without a version, read git tags the same way:

- **Only tags on the checked-out commit's history count**, so on a
  maintenance branch semvu sees that branch's versions, not newer ones from
  `main`. Add `--global` to read every tag in the repository.
- **`--for-major` keeps to one major version**, so `--for-major 1` reads
  `v1.4.2` and ignores `v2.0.0`.
- **Tags that aren't valid versions are skipped** (such as `nightly`), and any
  `v` is ignored.
- **The latest is chosen by semvu's ordering**, not by tag date or name.

These flags apply to both commands:

| Flag | Meaning |
|---|---|
| `--directory path` | Read the repository containing `path` instead of the current directory |
| `--global` | Read every tag, not only those on the checked-out commit's history |
| `--tag-prefix prefix` | Only read tags starting with `prefix`, and ignore the prefix: with `--tag-prefix api/`, the tag `api/v1.2.3` is version `1.2.3` |
| `--initial version` | Use `version` if no tag matches, as in a new repository, instead of failing |
| `--for-major major` | Only read tags with that MAJOR version: with `--for-major 1`, the tags `v1.4.2` and `v2.0.0` give `v1.4.2` |

### semvu latest

`semvu latest` (alias `semvu l`) prints the latest version in the git tags.

```sh
semvu latest                     # v1.3.0-rc-002    the latest version, pre-releases included
semvu latest --released          # v1.2.0           the latest release, skipping pre-releases
semvu latest --without-v         # 1.3.0-rc-002
semvu latest --global            # including tags on other branches
semvu latest --tag-prefix api/   # the latest api/… tag, as a version
semvu latest --for-major 1       # v1.4.2           the latest 1.x version, while v2.0.0 is out
semvu latest --initial 0.0.0     # 0.0.0 if there are no matching tags yet
semvu l --directory ../app       # the latest version in the repository at ../app
```

| Flag | Meaning |
|---|---|
| `--released` | Skip pre-releases |
| `--at-head` | Only read tags on the checked-out commit itself. It fails if there are none, so a script can check whether a commit is tagged yet |
| `--with-v`, `--without-v` | Always, or never, print a leading `v`. Without either, the version is printed as the tag has it |

It also takes [the tag flags](#which-tags-semvu-reads). Without `--initial`, it's
an error if there is no repository, no tag that is a valid version, or, with
`--released`, no released version.

### semvu modify

`semvu modify` (alias `semvu m`) reads one version, applies any modifiers, and
prints the result.

#### Where the version comes from

| You run | semvu reads |
|---|---|
| `semvu modify 1.2.3` | The argument |
| `echo 1.2.3 \| semvu modify -` | Standard input, because the argument is `-` |
| `semvu modify` | The latest git tag, exactly as `semvu latest` finds it, using [the tag flags](#which-tags-semvu-reads) |
| `semvu modify --hotfix` | The latest *released* git tag |

semvu only reads standard input when you pass `-`. It never reads it otherwise,
so it can't hang in CI jobs, git hooks or loops where standard input is open
but not meant for semvu.

`--no-git` makes sure semvu never uses git: it needs a version, from the
argument or `-`, and fails with exit status `2` instead of falling back to the
git tags. Flags that read or make tags (the tag flags, `--auto`,
`--skip-release`, `--apply` and `--apply-annotate`) can't be used with it.

```sh
semvu modify "$VERSION" --patch --no-git   # fails, rather than reading the tags, if $VERSION is empty
```

#### Modifiers

Use at most one version modifier and at most one stage modifier.

| Modifier | Effect | Example |
|---|---|---|
| `--major` | Bump MAJOR, reset MINOR and PATCH to 0 | `1.2.3` → `2.0.0` |
| `--minor` | Bump MINOR, reset PATCH to 0 | `1.2.3` → `1.3.0` |
| `--patch` | Bump PATCH | `1.2.3` → `1.2.4` |
| `--hotfix` | Bump PATCH of the latest release, skipping pre-releases | tags `v1.2.3`, `v1.3.0-rc-001` → `1.2.4` |
| `--auto` | Work out the bump from the [Conventional Commits](#bumping-from-commit-messages) since the latest tag | `fix:` and `feat:` since `v1.2.3` → `1.3.0` |
| `--alpha`, `--beta`, `--rc` | Move to that stage at `001`, or to the next index if already there | `1.2.3-alpha-099` → `1.2.3-rc-001`<br>`1.2.3-rc-001` → `1.2.3-rc-002` |
| `--release` | Remove the stage from a pre-release | `1.2.3-rc-001` → `1.2.3` |

- A version modifier **on its own removes any stage**: `1.2.3-rc-004 --patch`
  gives `1.2.4`. To release `1.2.3-rc-004` as `1.2.3`, use `--release`.
- A version modifier **with a stage modifier** bumps the version and starts
  that stage at `001`: `1.2.3 --minor --alpha` gives `1.3.0-alpha-001`.
- **From git tags, a version modifier won't [skip a release](#skipping-a-release).**
  With `v1.2.3-rc-001` the latest tag, `--minor` fails until `v1.2.3` is
  tagged. Add `--skip-release` to move on anyway.
- **`--hotfix`** is for fixing what's already out while a pre-release is in
  progress. From git tags, it reads the latest *release*, so `v1.3.0-rc-001`
  doesn't get in the way. `--patch` there would fail, because `1.3.0` hasn't
  been released. Given a version directly, it must be a release.
- **`--for-major`** works on an older major version: with `v1.4.2` and
  `v2.0.0` tagged, `semvu modify --for-major 1 --patch` gives `v1.4.3`. The
  result must stay in that major version, so `--major` can't be used with it,
  and `--auto` fails (exit status `4`) if the commits contain a breaking
  change.
- With **no modifiers**, semvu prints the version as it is. That's useful for
  checking or normalising one (`v1.2.3` → `1.2.3`).

#### Bumping from commit messages

If your commit messages follow [Conventional Commits](https://www.conventionalcommits.org),
`--auto` works out the bump for you from the commits since the latest tag:

| Commits since the latest tag include | Bump |
|---|---|
| A breaking change: `!` before the colon, as in `feat!: …`, or a `BREAKING CHANGE:` footer | MAJOR (MINOR before 1.0.0) |
| `feat: …` | MINOR |
| `fix: …` or `perf: …` | PATCH |
| Only other types, such as `docs:` or `chore:`, or messages that don't follow the convention | None: semvu fails with exit status `8` |

```sh
semvu modify --auto                 # v1.3.0, for a feat since v1.2.3
semvu modify --auto --rc            # v1.3.0-rc-001
semvu modify --auto --explain       # the same, saying why on stderr:
# 4 commits since v1.2.3: 1 feat, 1 fix, 1 docs and 1 not following Conventional Commits.
# That's a minor bump, from 1.2.3 to 1.3.0.
```

- **Before 1.0.0**, a breaking change bumps MINOR, because anything may change
  in `0.y.z`. Moving to 1.0.0 is a deliberate step: use `--major`.
- **When the latest tag is a pre-release**, semvu checks whether it already
  covers the bump. If `v1.3.0-rc-001` is out and only fixes have landed
  since, `--auto --rc` gives `1.3.0-rc-002`, and `--auto` alone releases
  `1.3.0`. A breaking change would need `2.0.0-rc-001`, which
  [skips the release](#skipping-a-release) of `1.3.0`, so semvu fails unless
  you add `--skip-release`.
- Merge commits are skipped. With `--tag-prefix`, commits are counted from the
  prefixed tag, but include changes anywhere in the repository.
- `--auto` reads tags, so it takes [the tag flags](#which-tags-semvu-reads) and
  can't be used with a version. `--initial` works too, to count every commit
  in a repository with no tags yet.

#### Tagging the new version

`--apply` tags the checked-out commit with the new version, as `git tag` does.
`--apply-annotate` makes an annotated tag instead, as `git tag --annotate`
does: git opens your editor for the tag's message.

```sh
semvu modify --patch --apply              # prints v1.2.4, and tags v1.2.4
semvu modify --auto --rc --apply-annotate # opens your editor, then tags v1.3.0-rc-001
git push origin v1.2.4                  # semvu never pushes
```

- **The tag is exactly what's printed**, after any `--tag-prefix`. Read from
  the tag `v1.2.3`, the new version and tag are `v1.2.4`; with
  `--tag-prefix api/`, the tag is `api/v1.2.4`. Given a version as an
  argument, its `v` (or lack of one) decides, unless you pass `--with-v` or
  `--without-v`.
- **It never tags a version twice.** If the version is already tagged, in
  either style, semvu tags nothing, prints nothing, and exits with status `9`.
- **It works with every way of choosing the version**, including a version
  given as an argument, which is tagged in the current directory's
  repository.
- **stdout still holds only the version.** The editor uses the terminal
  through stdin and stderr, so `next="$(semvu m --patch --apply-annotate)"`
  works.
- If the editor fails, or you leave the message empty, git doesn't make the
  tag. semvu exits with status `7`. git picks the editor from `GIT_EDITOR`, the
  `core.editor` setting, `VISUAL` or `EDITOR`.

#### The `v`

A printed version keeps the `v` of the version semvu read, or its lack of one:

| semvu read | `v`? |
|---|---|
| the argument `v1.2.3`, or `v1.2.3` on standard input | yes |
| the tag `v1.2.3`, or `api/v1.2.3` with `--tag-prefix api/` | yes |
| `--initial v0.0.0`, when no tag matches | yes |
| `1.2.3` from any of those | no |

`--with-v` always adds one, and `--without-v` always leaves it out. You can't
give both. `semvu latest` follows the same rule.

Stages only move forward, as set out in
[`conventions/semver.md`](conventions/semver.md#stage-progression). semvu
refuses to:

- release a version that is already released (`1.2.3 --release`)
- put a release back into a stage (`1.2.3 --alpha`). Start a new version
  instead, as in `--patch --alpha`.
- go back to an earlier stage (`1.2.3-rc-001 --alpha`), unless you add
  `--allow-stage-regression`
- go past index `999`

#### Going back a stage

If a release candidate turns out to need more work, `--allow-stage-regression`
lets `--alpha`, `--beta` or `--rc` move a pre-release back to an earlier stage:

```sh
semvu modify 1.2.3-rc-001 --alpha --allow-stage-regression   # 1.2.3-alpha-001
semvu modify --alpha --allow-stage-regression                # v1.2.3-alpha-002, with v1.2.3-alpha-001 and v1.2.3-rc-001 tagged
```

- **From git tags, it carries on after the highest index already tagged** in
  that stage, on any branch, so it never repeats a version. Given a version
  directly, there are no tags to check, so the stage starts at `001`.
- **The latest version doesn't change.** `v1.2.3-alpha-002` still ranks below
  `v1.2.3-rc-001`, so the next `semvu modify --alpha` fails as usual. Add
  `--allow-stage-regression` again each time.
- **A release still can't go back into a stage**, and the flag only works with
  `--alpha`, `--beta` or `--rc`, not with a version modifier.

#### Skipping a release

A pre-release leads to its release. Reading from git tags, semvu won't start a
new version until the latest pre-release has been released, so a release can't
be skipped by accident:

```sh
semvu latest                       # v1.2.3-rc-001
semvu modify --minor --rc          # fails with exit status 4: "1.2.3" has not been released yet
semvu modify --release             # v1.2.3, so release it first
semvu modify --minor --rc --skip-release   # v1.3.0-rc-001, or skip it on purpose
```

- **It applies to `--major`, `--minor`, `--patch` and `--auto`.** Stage
  modifiers stay within the same version, and `--hotfix` starts from a
  release, so they never skip one.
- **A release tagged on any branch counts**, with the same `--tag-prefix`, as
  when `v1.2.3` was tagged on a release branch.
- **Given a version directly, there are no tags to check**, so
  `semvu modify 1.2.3-rc-001 --minor` gives `1.3.0`, and `--skip-release` can't
  be used.

Other flags:

| Flag | Meaning |
|---|---|
| `--with-v`, `--without-v` | Always, or never, print a leading `v` (see [the `v`](#the-v)) |
| `--allow-stage-regression` | Let `--alpha`, `--beta` or `--rc` [go back to an earlier stage](#going-back-a-stage) |
| `--skip-release` | Let `--major`, `--minor`, `--patch` or `--auto` [move on from an unreleased pre-release](#skipping-a-release) |
| `--no-git` | [Never use git](#where-the-version-comes-from): a version must be given |
| `--explain` | With `--auto`, say on stderr how the bump was chosen |
| `--apply` | Also [tag the checked-out commit](#tagging-the-new-version) with the new version |
| `--apply-annotate` | The same, as an annotated tag, writing its message in your editor |

With no version given, it also takes [the tag flags](#which-tags-semvu-reads).
They can't be used with a version, because there are no tags to read.

### semvu validate

`semvu validate` checks that a version is valid. It prints nothing: the exit
status is the answer, `0` if valid and `3` if not, with the reason on
standard error.

```sh
semvu validate 1.2.3-rc-001                    # exit status 0
semvu validate 1.2.3-alpha-1                   # exit status 3, and the reason
echo "$TAG" | semvu validate -                 # read the version from standard input
semvu validate "$TAG" 2>/dev/null && echo ok   # just the answer
```

### Other commands

```sh
semvu help [command]        # general help, or help for one command, on stdout (also just semvu)
semvu <command> --help      # the same help, on stderr (also -h)
semvu --version             # semvu's own version, as in 0.1.0
semvu version               # semvu's version with its build information
semvu completion <shell>    # a completion script; see Setup
semvu validate <version>    # check a version, by exit status; see above
```

Every command accepts `--colour auto|always|never` (alias `--color`).

## Recipes

### A release-candidate cycle

```sh
semvu latest                         # v1.2.3          what's out now

semvu m --minor --rc --apply         # tags v1.3.0-rc-001
semvu m --rc --apply                 # tags v1.3.0-rc-002
semvu m --release --apply            # tags v1.3.0
```

Each `semvu m` reads the tag created by the line before, so every step builds on
the last.

### Tag the next patch release in CI

```sh
# If this commit already has a version tag, as when a job is re-run, stop.
status=0
semvu latest --at-head > /dev/null 2>&1 || status=$?
if [ "$status" -eq 0 ]; then echo "Already tagged."; exit 0; fi
if [ "$status" -ne 6 ]; then exit "$status"; fi   # 6: no version tag yet

next="$(semvu modify --patch --apply)" || exit 1
git push origin "$next"
```

Without the check, re-running the job would tag the same commit a second
time, with the next version after that.

### Check a version before using it

```sh
if semvu validate "$INPUT" 2>/dev/null; then
  echo "valid"
else
  echo "not a valid semvu version: $INPUT"
fi
```

### Release whatever the commits call for

```sh
# Exit status 8 means no commit since the latest tag needs a release.
status=0
next="$(semvu modify --auto --apply)" || status=$?
if [ "$status" -eq 8 ]; then echo "Nothing to release."; exit 0; fi
if [ "$status" -ne 0 ]; then exit "$status"; fi

git push origin "$next"
```

### Hotfix a release while a release candidate is in progress

```sh
semvu latest                         # v1.3.0-rc-002    the rc in progress
semvu latest --released              # v1.2.3           what's out now

git switch -c hotfix/1.2 v1.2.3      # fix what's out, without the rc's changes
git commit -am "fix: …"              # the fix itself
semvu m --hotfix --apply             # tags v1.2.4
```

Branch from the release first: `--apply` tags the checked-out commit, so on
`main` the hotfix would include the rc's unreleased work. To work out the
number without branching, as in a planning step, `semvu m --hotfix --global`
gives `1.2.4` from anywhere.

### Start versioning a new repository

```sh
semvu m --initial v0.0.0 --minor --apply   # tags v0.1.0 when there are no tags yet
```

With tags present, `--initial` is ignored, so the same line works on every
run.

### Version one package in a monorepo

```sh
semvu m --tag-prefix api/ --minor --apply   # api/v1.4.0 → tags api/v1.5.0
```

### Work on a maintenance branch

```sh
git switch support/1.x
semvu latest                         # v1.2.5   this branch's latest, not main's v2.0.0
semvu m --patch --apply              # tags v1.2.6
```

### Patch an older major version

```sh
semvu latest                         # v2.0.0
semvu m --for-major 1 --patch        # v1.4.3   the next 1.x patch, from v1.4.2
```

To tag it, check out the commit to fix first, as for a hotfix: `--apply` tags
the checked-out commit.

### Version a project in another directory

```sh
semvu latest --directory ../api --released
```

## Scripting

semvu is built to be captured with `$(…)`:

- **stdout** holds the command's main output and nothing else: one version and
  a newline for `semvu latest` and `semvu modify`. It never contains colour, and
  it is **empty whenever semvu fails**. (`semvu help`, and `semvu` on its own,
  write help there too, so `semvu help | less` works.)
- **stderr** gets everything else: errors, hints, and help from `--help`.
- **Exit status** tells you what went wrong without reading stderr:

| Status | Meaning |
|---|---|
| `0` | Success |
| `1` | Internal failure, such as a bug; please report it |
| `2` | Usage error: unknown command or flag, bad flag value, conflicting flags |
| `3` | Invalid version |
| `4` | The version can't be modified as asked, such as going back a stage or skipping a release |
| `5` | No git repository where semvu looked |
| `6` | No tag that is a version of the kind asked for |
| `7` | git isn't installed, or failed |
| `8` | Nothing to release: `semvu modify --auto` found no commits that need a new version |
| `9` | The tag already exists: `semvu modify --apply` was asked to tag a version that's already tagged |

So this either gives a valid version, or stops the script with an empty
variable:

```sh
next="$(semvu modify --patch)" || exit 1
```

And this starts from scratch only when there really are no matching tags:

```sh
latest="$(semvu latest)"; status=$?
if [ "$status" -eq 6 ]; then latest=0.0.0; elif [ "$status" -ne 0 ]; then exit "$status"; fi
```

(`semvu latest --initial 0.0.0` does the same in one step.)

Colour is used only when the stream is a terminal. Set
[`NO_COLOR`](https://no-color.org) or pass `--colour=never` to turn it off, or
`--colour=always` (or `FORCE_COLOR`) to force it on.
[`conventions/output.md`](conventions/output.md) has the full rules.

## Development

You need:

- **Go**, at the version in `.go-version`. semvu always uses the latest Go
  release.
- [golangci-lint](https://golangci-lint.run) v2, built with that Go version.
- [GoReleaser](https://goreleaser.com) v2, if you want to build release
  artefacts locally.

### Set up Go with goenv

We recommend [goenv](https://github.com/go-nv/goenv) for managing Go. It reads
`.go-version` and switches to that version whenever you're in this repository,
so you're always on the version semvu builds with, whatever else you have
installed.

```sh
goenv install --skip-existing   # install the Go version in .go-version
go version                      # check: it should match .go-version
```

When semvu moves to a new Go release, pull and run the same `goenv install`
again.

Without goenv, install the version in `.go-version` some other way. Go's own
toolchain switching (`GOTOOLCHAIN=auto`, the default) will also fetch it for
you, as long as your installed Go is 1.21 or newer.

### Make targets

```sh
make help              # list all targets
make build             # build ./bin/semvu
make test              # run tests with the race detector, in random order
make cover-check       # run tests, and fail if coverage is below 95% (CI runs this)
make fuzz              # fuzz the version and commit parsers, 30 seconds each (CI runs this)
make cover             # run tests and open a coverage report
make lint              # run golangci-lint
make vuln              # check for known vulnerabilities that semvu's code can reach
make go-latest         # move go.mod and .go-version to the latest Go release
make go-version-check  # check .go-version matches go.mod
make check             # tidy, format, lint and test: run this before pushing
make snapshot          # build every release artefact into ./dist without publishing
make archive-check     # build from a source tarball, as Homebrew does, and check semvu knows its commit
```

### Project layout

```text
cmd/semvu/          Entry point. Wires OS streams and signals into the CLI; no logic.
internal/cli/       Cobra commands. Parses input, calls domain code, formats output.
internal/buildinfo/ Version, commit and build date of the running binary.
internal/semver/    Parsing, ordering, formatting and modifying semvu versions.
internal/commits/   Reads Conventional Commits messages and works out the bump.
internal/git/       Reads tags and commit messages by running the git command.
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
suite, then builds and publishes everything with GoReleaser. semvu can work out
the tag:

```sh
next="$(semvu modify --minor --apply-annotate)"   # opens your editor; tags v0.2.0, say
git push origin "$next"
```

Tags with a stage (for example `v0.2.0-rc-001`) are published as GitHub
pre-releases.

Before anything is published, the release workflow builds semvu from the tagged
commit and uses it to check the tag
([`scripts/check-release-tag.sh`](scripts/check-release-tag.sh)). The tag
must be a valid semvu version, and the latest version on its own history. So
`v0.2.0-rc.1` is refused, as is `v0.1.5` after `v0.2.0` on `main`, but a
hotfix such as `v0.1.5` on a maintenance branch is fine.

Each release is published with signed build provenance for every archive and
package (see [`SECURITY.md`](SECURITY.md#checking-a-download)).

## Licence

[MIT](LICENSE)
