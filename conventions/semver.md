# svm SemVer conventions

svm accepts a **strict subset** of [Semantic Versioning 2.0.0][semver]. Every
version svm accepts is a valid SemVer version once any leading `v` is removed.
The reverse is not true: many valid SemVer versions are rejected by svm.

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
| `STAGE` | One of `alpha`, `beta`, `rc` or `ga`. Lowercase only. |
| `INDEX` | Exactly three digits, zero-padded, from `001` to `999`. |

The `-STAGE-INDEX` part is called the **suffix**. Hyphens are the only
separators allowed inside it.

If a version has a suffix, it **must** have both a stage and an index.

Whole-version pattern, for reference:

```text
^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta|rc|ga)-([0-9]{3}))?$
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
| 4 | `ga` | General availability |
| 5 | *(none)* | Release version |

### Stage progression

For a given `MAJOR.MINOR.PATCH`:

- A version **may** skip any stage. For example, `alpha` can go straight to
  `rc`, `ga` or the release, and the first pre-release can be any stage.
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
valid successor to `1.2.3-rc-004`.

## Precedence

Versions are compared in this order:

1. `MAJOR`, `MINOR` and `PATCH`, numerically, as in SemVer §11.
2. Stage, in the order in [Stages](#stages). A release version ranks above every
   pre-release of the same `MAJOR.MINOR.PATCH`.
3. Index, numerically.

The `v` prefix is ignored. `v1.2.3` and `1.2.3` are equal.

```text
1.2.3-alpha-001 < 1.2.3-alpha-002 < 1.2.3-beta-001 < 1.2.3-rc-001
               < 1.2.3-ga-001 < 1.2.3 < 1.2.4-alpha-001
```

> [!WARNING]
> **This differs from SemVer precedence for `rc` and `ga`.**
> In SemVer, `-rc-001` and `-ga-001` are each a single alphanumeric pre-release
> identifier, and those are compared in ASCII order (SemVer §11.4). Because `g`
> comes before `r`, other SemVer tools rank `1.2.3-ga-001` *below*
> `1.2.3-rc-001`. That includes Go modules, `sort -V` and most package
> registries.
>
> svm ranks `ga` above `rc`. `alpha` < `beta` < `rc` is the same in both
> systems.

## Examples

### Valid

| Version | Notes |
|---|---|
| `1.2.3` | Release |
| `v1.2.3` | Release, with prefix |
| `0.0.0` | Zero is allowed |
| `1.2.3-alpha-001` | First alpha |
| `v2.3.4-rc-987` | Prefix and suffix together |
| `10.20.30-ga-999` | Highest index allowed |

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
| `1.2.3-001` | Index without a stage |
| `1.2.3-rc-001-002` | More than one stage and index |
| `1.2.3+build.5` | Build metadata isn't supported |

## Open questions

These aren't settled yet. Until they are, the answers above are the working
assumption.

- **Prefix on output.** svm accepts both `v1.2.3` and `1.2.3`. It isn't decided
  whether output keeps the input's form, always drops the `v`, or uses a flag.
- **Build metadata.** SemVer allows `+metadata`, for example `1.2.3+sha.abc`.
  It is rejected for now.
- **`ga` and release versions.** Both mean "generally available". It isn't
  decided whether `1.2.3-ga-002` to `1.2.3` is a normal path, or whether `ga`
  versions are usually final.

[semver]: https://semver.org/spec/v2.0.0.html
[rfc2119]: https://www.rfc-editor.org/rfc/rfc2119
