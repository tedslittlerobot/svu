# semvu SemVer conventions

semvu accepts a **strict subset** of [Semantic Versioning 2.0.0][semver]. Every
version semvu accepts is a valid SemVer version once any leading `v` is removed.
The reverse is not true: many valid SemVer versions are rejected by semvu.

In this document, **must**, **must not** and **may** are used as defined in
[RFC 2119][rfc2119].

## Format

```text
[v]MAJOR.MINOR.PATCH[-STAGE-INDEX]
```

| Part | Rule |
|---|---|
| `v` | Optional prefix. Lowercase only. |
| `MAJOR`, `MINOR`, `PATCH` | Non-negative integers with no leading zeros, as in SemVer §2. `0` is allowed. `01` is not. |
| `STAGE` | One of `alpha`, `beta` or `rc`. Lowercase only. |
| `INDEX` | Exactly three digits, zero-padded, from `001` to `999`. |

The `-STAGE-INDEX` part is called the **suffix**. Hyphens are the only
separators allowed inside it.

If a version has a suffix, it **must** have both a stage and an index.

Build metadata (`+…`, SemVer §10) is **not** supported. Any version that
contains a `+` is invalid.

Whole-version pattern, for reference:

```text
^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta|rc)-([0-9]{3}))?$
```

The pattern can't express one extra rule: an index of `000` is invalid.

## Release versions

A **release version** has no suffix: `1.2.3` or `v1.2.3`.

A **pre-release version** has a suffix: `1.2.3-beta-002`.

## Stages

Stages mark how far along a version is. Their order is fixed:

| Order | Stage | Meaning |
|---|---|---|
| 1 | `alpha` | Early, incomplete, unstable |
| 2 | `beta` | Feature-complete, still being tested |
| 3 | `rc` | Release candidate |
| 4 | *(none)* | Release version |

Stage names are deliberately in alphabetical order. That keeps semvu's ordering
the same as plain SemVer ordering (see [Precedence](#precedence)). Any new stage
**must** fit into the alphabetical order where it belongs.

### Stage progression

For a given `MAJOR.MINOR.PATCH`:

- A version **may** skip any stage. For example, `alpha` can go straight to
  `rc` or the release, and the first pre-release can be any stage.
- A version **must not** go back to an earlier stage. Once `1.2.3` has reached
  `beta`, `1.2.3-alpha-…` can't be produced.
- Within a stage, the index **must** increase. The first index in a stage is
  `001`.
- When a version moves to a new stage, its index starts again at `001`.
- A stage allows at most 999 iterations. Going past `999` is an error. It never
  wraps round or moves the version into the next stage.
- Once a version is released, it can't go back into any stage. Pre-release work
  must start on a new `MAJOR.MINOR.PATCH`.

These rules only apply within one `MAJOR.MINOR.PATCH`. `1.2.4-alpha-001` is a
valid successor to `1.2.3-rc-004`, though it
[skips the release](#skipping-a-release) of `1.2.3`.

### Going back a stage

The rule that a version must not go back to an earlier stage has one
exception. When explicitly asked to (`semvu modify --allow-stage-regression`),
a pre-release **may** go back to an earlier pre-release stage of the same
`MAJOR.MINOR.PATCH`, as when a release candidate turns out to need more work.

- The index in the earlier stage **must** follow the highest index that
  version already has in that stage, so that no version is produced twice.
  With `1.2.3-alpha-001` and `1.2.3-rc-001` already out, going back to `alpha`
  gives `1.2.3-alpha-002`. If none is known, the index starts at `001`.
- Every other rule still applies. A release can't go back into any stage, and
  a stage allows at most 999 iterations.

Going back doesn't change [Precedence](#precedence): `1.2.3-alpha-002` still
ranks below `1.2.3-rc-001`, so the latest version stays the release candidate.

### Skipping a release

A pre-release leads to its release. Moving from a pre-release to a different
`MAJOR.MINOR.PATCH` **skips** that release: from `1.2.3-rc-001`, both
`1.3.0-rc-001` and `1.2.4` skip `1.2.3`. Moving within the same
`MAJOR.MINOR.PATCH`, to a later index, a later stage or the release itself,
never does.

- When semvu reads the version from git tags (`semvu modify` with no
  version), it **must not** skip a release that hasn't been tagged, unless
  explicitly asked to (`semvu modify --skip-release`). A release tagged on any
  branch counts.
- A version given directly has no tags to check, so it **may** skip a release.

## Precedence

Versions are compared in this order:

1. `MAJOR`, `MINOR` and `PATCH`, numerically, as in SemVer §11.
2. Stage, in the order in [Stages](#stages). A release version ranks above every
   pre-release of the same `MAJOR.MINOR.PATCH`.
3. Index, numerically.

The `v` prefix is ignored. `v1.2.3` and `1.2.3` are equal.

```text
1.2.3-alpha-001 < 1.2.3-alpha-002 < 1.2.3-beta-001 < 1.2.3-rc-001
               < 1.2.3 < 1.2.4-alpha-001
```

> [!NOTE]
> **For any two semvu versions, semvu's ordering is the same as SemVer's.**
> In SemVer, a suffix like `-alpha-001` is a single alphanumeric pre-release
> identifier, and these are compared in ASCII order (SemVer §11.4). semvu's stage
> names sort alphabetically and its indexes have a fixed width, so ASCII order
> and semvu order always agree.
>
> This means other SemVer tools sort semvu versions correctly, including Go
> modules, `sort -V` and package registries. Keep it that way: don't add a stage
> that would break the alphabetical order, or change the width of the index.

## Examples

### Valid

| Version | Notes |
|---|---|
| `1.2.3` | Release |
| `v1.2.3` | Release, with prefix |
| `0.0.0` | Zero is allowed |
| `1.2.3-alpha-001` | First alpha |
| `v2.3.4-rc-987` | Prefix and suffix together |
| `10.20.30-beta-999` | Highest index allowed |

### Invalid

| Version | Why |
|---|---|
| `1.2` | Missing patch |
| `01.2.3` | Leading zero in a version number |
| `V1.2.3` | Uppercase prefix |
| `1.2.3-alpha` | Suffix has no index |
| `1.2.3-alpha-1` | Index isn't three digits |
| `1.2.3-alpha-0001` | Index has more than three digits |
| `1.2.3-alpha-000` | Index is below `001` |
| `1.2.3-alpha.001` | Uses `.` as the suffix separator, not `-` |
| `1.2.3-alpha001` | No separator between stage and index |
| `1.2.3-Alpha-001` | Uppercase stage |
| `1.2.3-dev-001` | Unknown stage |
| `1.2.3-ga-001` | Unknown stage (`ga` isn't a stage) |
| `1.2.3-001` | Index without a stage |
| `1.2.3-rc-001-002` | More than one stage and index |
| `1.2.3+build.5` | Build metadata isn't supported |
| `1.2.3-rc-001+build.5` | Build metadata isn't supported |

## Out of scope

This document only defines which versions are valid, how they're ordered and
how stages progress. How semvu prints versions, including when it adds a `v`
prefix, is set out in [semvu's output conventions](output.md).

[semver]: https://semver.org/spec/v2.0.0.html
[rfc2119]: https://www.rfc-editor.org/rfc/rfc2119
