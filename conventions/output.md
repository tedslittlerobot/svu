# semvu output conventions

semvu is a tool for developers, and it's mostly run inside scripts, CI jobs and
shell pipelines. Its output has two jobs:

1. **Machines** need a command's main output, clean enough to capture without
   parsing.
2. **People** need errors and help they can understand at a glance.

semvu uses stdout for the first job and stderr for the second, and never mixes
them.

In this document, **must**, **must not**, **should** and **may** are used as
defined in [RFC 2119][rfc2119].

## stdout: a command's main output, and nothing else

Every command has one **main output**: the thing it exists to produce.
stdout **must** contain that and nothing else.

| Command | Main output |
|---|---|
| `semvu modify` | The modified version, valid under [semvu's version rules](semver.md) |
| `semvu latest` | The latest version in the repository's git tags |
| `semvu validate` | Nothing: the exit status is the answer |
| `semvu --version` | semvu's own version, bare, such as `1.4.0` |
| `semvu version` | semvu's version with its build information, on one line |
| `semvu completion <shell>` | The completion script, as in `source <(semvu completion bash)` |
| `semvu help [command]` | Help, so that `semvu help \| less` works |
| `semvu` or `semvu completion`, with no command | Help, the same as `semvu help` |

Help asked for with the `--help` or `-h` flag is not a main output: the flag
interrupts some other command. It goes to stderr. Help that a command prints
because it was asked for, by `semvu help` or by running a command group on its
own, goes to stdout.

- On success, stdout **must** contain the main output followed by a single
  newline (`\n`). For `semvu modify` and `semvu latest`, that is exactly one
  version.
- A version is printed with a lowercase `v` if semvu read it with one: from the
  argument, standard input, the tag (after any `--tag-prefix`) or
  `--initial`. `--with-v` always adds one and `--without-v` always leaves it
  out; they can't be used together.
- stdout **must not** contain anything else: no labels, prompts, progress
  messages, warnings, blank lines or trailing whitespace.
- stdout **must not** contain ANSI escape codes, even when it is a terminal.
  The one exception is help, whose output is for people: it follows the
  [colour rules](#when-colour-is-used) for stdout, so it is plain whenever
  stdout isn't a terminal.
- On failure, stdout **must** be empty. semvu never prints a partial or
  best-guess result before failing. A command with a side effect, such as
  `semvu modify --apply`, does it before printing, so the output means it
  happened.

This means the following always gives either a valid version or a non-zero
exit status with an empty variable:

```sh
next="$(semvu modify …)" || exit 1
```

## stderr: everything else

stderr **must** be used for all output that isn't a command's main output.
That includes:

- errors and warnings
- help and usage text from `--help` or `-h`
- explanations a flag asks for, such as `semvu modify --auto --explain`. These
  are plain sentences with no label, and are only written when asked for
- the editor that `semvu modify --apply-annotate` opens through git, which uses
  the terminal through stdin and stderr, so that stdout still holds only the
  version
- progress for a slow step that a flag asks for, such as the push that
  `semvu modify --push` makes (see [Progress](#progress))
- any diagnostic or explanatory message

## Exit status

Every kind of failure has its own exit status, so scripts can tell them apart
without reading stderr. The numbers are part of semvu's interface and **must
not** change.

| Status | Meaning | Examples |
|---|---|---|
| `0` | Success | |
| `1` | Internal failure that the user can't fix | A bug; output couldn't be written; standard input couldn't be read |
| `2` | Usage error | Unknown command or flag; bad flag value; flags that can't be used together; too many arguments |
| `3` | Invalid version | An argument, standard input or `--initial` isn't a valid version |
| `4` | The version can't be modified as asked | Going back a stage; releasing a release; hotfixing a pre-release; a number too large to bump; `--auto` needing a new major version with `--for-major`; skipping the release of a pre-release in the git tags |
| `5` | No repository | Not in a git repository; `--directory` doesn't exist or isn't a directory |
| `6` | No matching tag | No tag is a valid version, or none is a release when only releases are wanted |
| `7` | git failed | git isn't installed, or reported an error |
| `8` | Nothing to release | `semvu modify --auto` found no commits since the latest tag that need a new version |
| `9` | Tag already exists | `semvu modify --apply` was asked to tag a version that is already tagged |
| `10` | No remote to push to | `semvu modify --push` found no remotes, several and none called `origin`, or not the one `--remote` names |

A new kind of failure gets a new number. A failure that fits an existing kind
uses its number.

## Writing messages

Every message on stderr is written for a person to read.

### Language

- Messages **must** be in clear, correctly punctuated **British English**:
  "colour", "behaviour", "recognised", "initialise", "licence" (noun).
- Every sentence **must** start with a capital letter and end with a full stop.
- Use plain, direct words. Say what's wrong; don't describe what semvu was doing
  when it went wrong.
- Never blame the user, apologise, or use exclamation marks.
- Values the user gave **must** be quoted with straight double quotes, for
  example `"1.2.3-alpha-1"`. Flags and commands are written as typed, for
  example `--colour`. Neither is translated or reworded.
- Internal details **must not** reach the user: no Go error chains such as
  `parse: strconv.Atoi: …`, no stack traces, no type names, no `%v` dumps.

### Errors

An error has a **headline**. It should also have **detail** and a **hint**
whenever semvu can offer them:

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
| Detail | **Should** explain the rule that was broken, in terms of [semvu's version rules](semver.md). Indented by two spaces. |
| Hint | **Should** be given when semvu can suggest a fix. Starts with `Hint: `. When semvu can work out the corrected value, the hint should include it. |

Keep it short. One headline, a few lines of detail at most, and at most one
hint. If more than one thing is wrong, report the first problem clearly rather
than all of them at once.

An unexpected failure that the user can't fix (a bug, or an I/O failure)
still gets a proper headline, and asks for a report:

```text
Error: semvu could not write its output.

  The output stream was closed before semvu finished writing.

Hint: If this keeps happening, please report it at https://github.com/tedslittlerobot/semvu/issues.
```

### Progress

A step that can take a while, such as a push to a remote, says on stderr
when it starts and when it has finished, one line each:

```text
Pushing v1.2.4 to origin...
Pushed v1.2.4 to origin.
```

- The line that starts a step ends with `...` instead of a full stop. The
  line that finishes it is an ordinary sentence.
- Names, such as a tag or remote, are written as they are, not quoted, and
  shown in bold.
- If the step fails, there is no finishing line. The error follows, and says
  what state the step left behind and how to finish it, as in
  `git push origin v1.2.4`.
- Progress is written whether or not stderr is a terminal, so it shows in CI
  logs too.

### Warnings

Warnings follow the same rules as errors but start with `Warning: `. A warning
never changes what goes to stdout, and it never makes the exit status
non-zero.

## Colour

Colour is used on **stderr**, and for help on stdout. It makes messages
quicker to scan, but must never be needed to understand them. Every message
has to make complete sense as plain text.

### When colour is used

semvu decides whether to use colour on a stream in this order. The first rule
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
| Progress verb for a step in progress, such as `Pushing` | Bold cyan |
| Progress verb for a finished step, such as `Pushed` | Bold green |
| Names in a progress line, such as a tag or remote | Bold |
| Commands, flags and values suggested in a hint | Green |
| Detail text | Default colour |
| Secondary information (paths, URLs) | Dim |
| Help section headings | Bold |
| Command and flag names in help, including flags named in descriptions | Cyan |
| Placeholders, usage arguments and example comments in help | Dim |

- Use only the 16 basic ANSI colours, plus bold and dim, so that the user's
  terminal theme decides the actual shades. No 256-colour or truecolour codes.
- Reset styling at the end of every styled span. semvu **must** never leave the
  terminal in a styled state, even when it is interrupted.

## Implementation notes

- **Go errors vs messages.** Go error values inside semvu follow normal Go style:
  lower case, no full stop, wrapped with `%w`. They are never printed as they
  are. The `internal/cli` layer is the only place that turns errors into the
  user-facing messages described here. Domain packages return typed errors
  that carry what the CLI needs, such as the input, the position and the rule
  that was broken.
- **Marking suggestions.** In a hint, suggested commands, flags and values go
  in backticks, as in ``run `semvu --help` ``. They're shown green with colour
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
