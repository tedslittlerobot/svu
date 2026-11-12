# Security policy

## Supported versions

Security fixes go into the latest release only. svu is before v1.0.0, so a
fix ships as a new release rather than being backported to older ones.

## Reporting a vulnerability

Please **don't** report a security problem in a public issue, a pull request
or a discussion.

Report it privately through GitHub instead: on the repository's
[Security tab](https://github.com/tedslittlerobot/svu/security), choose
**Report a vulnerability**, or go straight to the
[new advisory form](https://github.com/tedslittlerobot/svu/security/advisories/new).

It helps to include:

- the svu version, from `svu version`
- what you ran, and what happened
- why it's a security problem, and what an attacker could do with it

The report stays private while it's looked into. Once a fix is released, the
advisory is published, with credit to you if you'd like it.

## Checking a download

Each release comes with signed build provenance, a record that each archive
and package was built by this repository's release workflow. To check a file
you've downloaded, with the [GitHub CLI](https://cli.github.com):

```sh
gh attestation verify svu_0.1.0_linux_amd64.tar.gz --repo tedslittlerobot/svu
```

Each release also publishes `checksums.txt`. The
[installation instructions](README.md#installation) show how to check a
download against it.
