# svu output conventions

svu is a tool for developers, and it's mostly run inside scripts, CI jobs and
shell pipelines. Its output has two jobs:

1. **Machines** need a command's main output, clean enough to capture without
   parsing.
2. **People** need errors and help they can understand at a glance.

svu uses stdout for the first job and stderr for the second, and never mixes
them.

In this document, **must**, **must not**, **should** and **may** are used as
defined in [RFC 2119][rfc2119].

## stdout: a command's main output, and nothing else

Every command has one **main output**: the thing it exists to produce.
stdout **must** contain that and nothing else.

| Command | Main output |
|---|---|
| `svu modify` | The modified version, valid under [svu's version rules](semver.md) |
| `svu latest` | The latest version in the repository's git tags |
| `svu validate` | Nothing: the exit status is the answer |
| `svu --version` | svu's own version, bare, such as `1.4.0` |
| `svu version` | svu's version with its build information, on one line |
| `svu completion <shell>` | The completion script, as in `source <(svu completion bash)` |
| `svu help [command]` | Help, so that `svu help \| less` works |
| `svu` or `svu completion`, with no command | Help, the same as `svu help` |

Help asked for with the `--help` or `-h` flag is not a main output: the flag
interrupts some other command. It goes to stderr. Help that a command prints
because it was asked for, by `svu help` or by running a command group on its
own, goes to stdout.

- On success, stdout **must** contain the main output followed by a single
  newline (`\n`). For `svu modify` and `svu latest`, that is exactly one
  version.
- A version is printed with a lowercase `v` if svu read it with one: from the
  argument, standard input, the tag (after any `--tag-prefix`) or
  `--initial`. `--with-v` always adds one and `--without-v` always leaves it
  out; they can't be used together.
- stdout **must not** contain anything else: no labels, prompts, progress
  messages, warnings, blank lines or trailing whitespace.
- stdout **must not** contain ANSI escape codes, even when it is a terminal.
  The one exception is help, whose output is for people: it follows the
  [colour rules](#when-colour-is-used) for stdout, so it is plain whenever
  stdout isn't a terminal.
- On failure, stdout **must** be empty. svu never prints a partial or
  best-guess result before failing. A command with a side effect, such as
  `svu modify --apply`, does it before printing, so the output means it
  happened.

This means the following always gives either a valid version or a non-zero
exit status with an empty variable:

```sh
next="$(svu modify …)" || exit 1
```

## stderr: everything else

stderr **must** be used for all output that isn't a command's main output.
That includes:

- errors and warnings
- help and usage text from `--help` or `-h`
- explanations a flag asks for, such as `svu modify --auto --explain`. These
  are plain sentences with no label, and are only written when asked for
- the editor that `svu modify --apply-annotate` opens through git, which uses
  the terminal through stdin and stderr, so that stdout still holds only the
  version
- any diagnostic or explanatory message

## Exit status

Every kind of failure has its own exit status, so scripts can tell them apart
without reading stderr. The numbers are part of svu's interface and **must
not** change.

| Status | Meaning | Examples |
|---|---|---|
| `0` | Success | |
| `1` | Internal failure that the user can't fix | A bug; output couldn't be written; standard input couldn't be read |
| `2` | Usage error | Unknown command or flag; bad flag value; flags that can't be used together; too many arguments |
| `3` | Invalid version | An argument, standard input or `--initial` isn't a valid version |
| `4` | The version can't be modified as asked | Going back a stage; releasing a release; hotfixing a pre-release; a number too large to bump |
| `5` | No repository | Not in a git repository; `--directory` doesn't exist or isn't a directory |
| `6` | No matching tag | No tag is a valid version, or none is a release when only releases are wanted |
| `7` | git failed | git isn't installed, or reported an error |
| `8` | Nothing to release | `svu modify --auto` found no commits since the latest tag that need a new version |
| `9` | Tag already exists | `svu modify --apply` was asked to tag a version that is already tagged |

A new kind of failure gets a new number. A failure that fits an existing kind
uses its number.

## Writing messages

Every message on stderr is written for a person to read.

### Language

- Messages **must** be in clear, correctly punctuated **British English**:
  "colour", "behaviour", "recognised", "initialise", "licence" (noun).
- Every sentence **must** start with a capital letter and end with a full stop.
- Use plain, direct words. Say what's wrong; don't describe what svu was doing
  when it went wrong.
- Never blame the user, apologise, or use exclamation marks.
- Values the user gave **must** be quoted with straight double quotes, for
  example `"1.2.3-alpha-1"`. Flags and commands are written as typed, for
  example `--colour`. Neither is translated or reworded.
- Internal details **must not** reach the user: no Go error chains such as
  `parse: strconv.Atoi: …`, no stack traces, no type names, no `%v` dumps.

### Errors

An error has a **headline**. It should also have **detail** and a **hint**
whenever svu can offer them:

```text
Error: "1.2.3-alpha-1" is not a valid version.

  1.2.3-alpha-1
              ^
  The stage index must be exactly three digits, from 001 to 999.

Hint: Did you mean "1.2.3-alpha-001"?
```

| Part | Rule |
|---|---|
| Headline | **Must** start with `Error: ` and be one sentence saying what went wrong, naming the input involved. |
| Pointer | **Should** be shown when the error is at a known position in one input. Show the input and a `^` under the problem. |
| Detail | **Should** explain the rule that was broken, in terms of [svu's version rules](semver.md). Indented by two spaces. |
| Hint | **Should** be given when svu can suggest a fix. Starts with `Hint: `. When svu can work out the corrected value, the hint should include it. |

Keep it short. One headline, a few lines of detail at most, and at most one
hint. If more than one thing is wrong, report the first problem clearly rather
than all of them at once.

An unexpected failure that the user can't fix (a bug, or an I/O failure)
still gets a proper headline, and asks for a report:

```text
Error: svu could not write its output.

  The output stream was closed before svu finished writing.

Hint: If this keeps happening, please report it at https://github.com/tedslittlerobot/svu/issues.
```

### Warnings

Warnings follow the same rules as errors but start with `Warning: `. A warning
never changes what goes to stdout, and it never makes the exit status
non-zero.

## Colour

Colour is used on **stderr**, and for help on stdout. It makes messages
quicker to scan, but must never be needed to understand them. Every message
has to make complete sense as plain text.

### When colour is used

svu decides whether to use colour on a stream in this order. The first rule
that applies wins:

1. `--colour=always` or `--colour=never`, if given. `--color` is accepted as an
   alias.
2. If [`NO_COLOR`][no-color] is set to any non-empty value, there's no colour.
3. If `FORCE_COLOR` is set to any non-empty value, colour is used.
4. If `TERM` is `dumb`, there's no colour.
5. Otherwise, colour is used only if the stream is a terminal.

`--colour=auto` (the default) means "use rules 2 to 5".

### Palette

Errors are mainly shown in **red**. Other colours are only for emphasis or to
make particular parts stand out.

Every error and every help page is styled, including errors about flags and
unknown commands. No message reaches stderr unstyled when colour is on.

| Element | Style |
|---|---|
| `Error:` label | Bold red |
| Error headline | Red |
| Quoted user input in the headline | Bold red |
| `^` pointer, and the input from the problem onwards | Red |
| `Warning:` label | Bold yellow |
| Warning headline | Yellow |
| `Hint:` label | Bold cyan |
| Commands, flags and values suggested in a hint | Green |
| Detail text | Default colour |
| Secondary information (paths, URLs) | Dim |
| Help section headings | Bold |
| Command and flag names in help, including flags named in descriptions | Cyan |
| Placeholders, usage arguments and example comments in help | Dim |

- Use only the 16 basic ANSI colours, plus bold and dim, so that the user's
  terminal theme decides the actual shades. No 256-colour or truecolour codes.
- Reset styling at the end of every styled span. svu **must** never leave the
  terminal in a styled state, even when it is interrupted.

## Implementation notes

- **Go errors vs messages.** Go error values inside svu follow normal Go style:
  lower case, no full stop, wrapped with `%w`. They are never printed as they
  are. The `internal/cli` layer is the only place that turns errors into the
  user-facing messages described here. Domain packages return typed errors
  that carry what the CLI needs, such as the input, the position and the rule
  that was broken.
- **Marking suggestions.** In a hint, suggested commands, flags and values go
  in backticks, as in ``run `svu --help` ``. They're shown green with colour
  on. The backticks are removed either way, so the plain text reads normally.
  Quoted values in a hint are green too.
- **One writer per stream.** Commands write their main output with
  `cmd.OutOrStdout()`.
  They don't write errors themselves; they return them, and the CLI renders
  them to stderr.
- **Tests.** Command tests assert stdout, stderr and the exit status exactly.
  Colour rendering is tested with colour forced on and forced off. Every error
  a user can trigger needs a test that checks its exact wording.

[rfc2119]: https://www.rfc-editor.org/rfc/rfc2119
[no-color]: https://no-color.org
