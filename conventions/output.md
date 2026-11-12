# svm output conventions

svm is a tool for developers, and it's mostly run inside scripts, CI jobs and
shell pipelines. Its output has two jobs:

1. **Machines** need a clean result they can capture without parsing.
2. **People** need errors they can understand and fix at a glance.

svm uses stdout for the first job and stderr for the second, and never mixes
them.

In this document, **must**, **must not**, **should** and **may** are used as
defined in [RFC 2119][rfc2119].

## stdout: the result, and nothing else

stdout is only for the result of a command, and for svm that result is a
version that is valid under [svm's version rules](semver.md).

- On success, stdout **must** contain exactly one version, followed by a single
  newline (`\n`).
- stdout **must not** contain anything else: no labels, prompts, progress
  messages, warnings, blank lines or trailing whitespace.
- stdout **must not** ever contain ANSI escape codes, even when it is a
  terminal.
- On failure, stdout **must** be empty. svm never prints a partial or
  best-guess result before failing.

This means the following always gives either a valid version or a non-zero
exit status with an empty variable:

```sh
next="$(svm …)" || exit 1
```

### Exceptions

These commands have outputs that other programs read, so they write to stdout:

| Output | Why it goes to stdout |
|---|---|
| `svm --version` | It prints svm's own version as a bare version, such as `1.4.0`, so it fits the rule above. |
| `svm completion <shell>` | Shells read completion scripts from stdout, as in `source <(svm completion bash)`. |

Nothing else is an exception. Help text, usage and the detailed output of
`svm version` all go to stderr.

## stderr: everything else

stderr **must** be used for all output that isn't the result. That includes:

- errors and warnings
- help and usage text, which is shown only for `--help` or `-h`
- detailed build information (`svm version`)
- any diagnostic or explanatory message

## Writing messages

Every message on stderr is written for a person to read.

### Language

- Messages **must** be in clear, correctly punctuated **British English**:
  "colour", "behaviour", "recognised", "initialise", "licence" (noun).
- Every sentence **must** start with a capital letter and end with a full stop.
- Use plain, direct words. Say what's wrong; don't describe what svm was doing
  when it went wrong.
- Never blame the user, apologise, or use exclamation marks.
- Values the user gave **must** be quoted with straight double quotes, for
  example `"1.2.3-alpha-1"`. Flags and commands are written as typed, for
  example `--colour`. Neither is translated or reworded.
- Internal details **must not** reach the user: no Go error chains such as
  `parse: strconv.Atoi: …`, no stack traces, no type names, no `%v` dumps.

### Errors

An error has a **headline**. It should also have **detail** and a **hint**
whenever svm can offer them:

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
| Detail | **Should** explain the rule that was broken, in terms of [svm's version rules](semver.md). Indented by two spaces. |
| Hint | **Should** be given when svm can suggest a fix. Starts with `Hint: `. When svm can work out the corrected value, the hint should include it. |

Keep it short. One headline, a few lines of detail at most, and at most one
hint. If more than one thing is wrong, report the first problem clearly rather
than all of them at once.

An unexpected failure that the user can't fix (a bug, or an I/O failure)
still gets a proper headline, and asks for a report:

```text
Error: svm could not write its output.

  The output stream was closed before svm finished writing.

Hint: If this keeps happening, please report it at https://github.com/tedslittlerobot/svm/issues.
```

### Warnings

Warnings follow the same rules as errors but start with `Warning: `. A warning
never changes what goes to stdout, and it never makes the exit status
non-zero.

## Colour

Colour is only used on **stderr**. It makes messages quicker to scan, but must
never be needed to understand them. Every message has to make complete sense
as plain text.

### When colour is used

svm decides whether to use colour on stderr in this order. The first rule that
applies wins:

1. `--colour=always` or `--colour=never`, if given. `--color` is accepted as an
   alias.
2. If [`NO_COLOR`][no-color] is set to any non-empty value, there's no colour.
3. If `FORCE_COLOR` is set to any non-empty value, colour is used.
4. If `TERM` is `dumb`, there's no colour.
5. Otherwise, colour is used only if stderr is a terminal.

`--colour=auto` (the default) means "use rules 2 to 5".

### Palette

Errors are mainly shown in **red**. Other colours are only for emphasis or to
make particular parts stand out.

| Element | Style |
|---|---|
| `Error:` label and headline | Bold red |
| `^` pointer and the problem part of the input | Red |
| Quoted user input in the headline | Bold, keeping the colour around it |
| `Warning:` label and headline | Bold yellow |
| `Hint:` label | Bold cyan |
| Suggested values in a hint | Green |
| Detail text | Default colour |
| Secondary information (paths, URLs) | Dim |
| Help section headings | Bold |
| Command and flag names in help | Cyan |
| Placeholders and example comments in help | Dim |

- Use only the 16 basic ANSI colours, plus bold and dim, so that the user's
  terminal theme decides the actual shades. No 256-colour or truecolour codes.
- Reset styling at the end of every styled span. svm **must** never leave the
  terminal in a styled state, even when it is interrupted.

## Implementation notes

- **Go errors vs messages.** Go error values inside svm follow normal Go style:
  lower case, no full stop, wrapped with `%w`. They are never printed as they
  are. The `internal/cli` layer is the only place that turns errors into the
  user-facing messages described here. Domain packages return typed errors
  that carry what the CLI needs, such as the input, the position and the rule
  that was broken.
- **One writer per stream.** Commands write results with `cmd.OutOrStdout()`.
  They don't write errors themselves; they return them, and the CLI renders
  them to stderr.
- **Tests.** Command tests assert stdout, stderr and the exit status exactly.
  Colour rendering is tested with colour forced on and forced off. Every error
  a user can trigger needs a test that checks its exact wording.

[rfc2119]: https://www.rfc-editor.org/rfc/rfc2119
[no-color]: https://no-color.org
