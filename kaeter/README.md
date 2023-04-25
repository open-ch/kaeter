# kætər

Welcome to `kaeter`:

> Caterpillars /ˈkætərˌpɪlər/ are the larval stage of members of the order Lepidoptera (the insect order comprising butterflies and moths).

This CLI tool has these goals:

- providing a standardised release process in a fat repo context
- tracking released versions
- providing a _just type this command to trigger a release_ interface to release anything

Basically, in most use cases, `kaeter` answers these two most common questions:

> How do I release this and what is the next version number?

## TL/DR

If you're only interested in quickly setting up a `kaeter` module in the context of Panta
without learning about the details, [directly jump to this how-to](../../doc/how-to/kaeter.md).

## History

`kaeter` is the original tool that lets you version and release arbitrary deliverables in a consistent way across a fat repo and/or an organisation.
It was built to be (and is still intended to remain so) useable in a standalone fashion outside of the present repository or OSAG.

With time, several features that where required in conjunction with `kaeter` but that did not belong directly with it where required:
they pertained to quality checks and wider integration with Bazel and the overall CI/CD:

 - quality checks: `kaeter-police` came in to enforce some rules around kaeter's usage. It was eventually refactored inside of kaeter as `kaeter lint`.
   Technically it could be the same binary called with other options.
 - CI, CD & Bazel integration: `kaeter-ci` emerged when we started to need some tooling to understand when a module needed, or didn't need, to be rebuilt.
   Its general purpose is to understand what has changed between two commits: it comes in handy to understand when to release modules but is not strictly linked to `kaeter`.

All in all, the _kaeter tooling_ family was (and still is) intended as something that helps you turn code into deliverables in a fat repo context.


## Instrumenting your project
`kaeter` currently works with Makefiles, in which it expects to find following targets:

- `build`
- `test`
- `release`
- `snapshot` (optional: your toolchain expects this, `kaeter` does not need it)

A `VERSION` environment variable set to the version being currently released will be passed to all targets when they are run,
as if you were calling  `make <target> -e VERSION=<version>`.

The `build` and `release` steps need to explicitly build //everything// that is required for the released module to be useable.

## Underlying principles

The base constraints are:

- Only code that currently exists on the remote master branch may be subject to a release.
- Release version numbers and anything required to identify a release is stored in git.
- Tags may be wiped completely from the repo: you may use them, but don't rely on them exclusively.

#### Release Identification

> Has this module, with the source code as currently present in git, already been released,
and if yes, under which version?

This question should always be answerable easily.

#### Immutable Source Identifier

Code to be released belongs to a commit that will never be updated:
**the commit id can identify the release**.

#### Release identity

Released versions are described in an unambiguous way and cannot be erased at will.
This description includes:

- Date of release
- Commit ID
- Version number

## Process

`kaeter` essentially follows these steps:

1. Someone asks to release module X, as it currently appears on origin/master
2. This results in a _Release Plan_ that identifies what is being released to be written to a commit message
3. This commit must be reviewed
4. After review and once pushed, this commit triggers a build step on a build host
5. The build host executes the release plan

## Content of a release plan

Currently, a release plan consists of a simple YAML array named `release`,
which contains an entry for each module to be released:

```yaml

releases:
  - groupId:ModuleId:version
  - nonMavenId:version

```

## How To

**Note**: references to kaeter below should be `./scripts/kaeter.sh` if you are at the top of the Panta repo.

### Initialise A Module

To initialise a module living at `my/module`

```shell
kaeter -p my/module init --id com.domain.my:my-module-id
```

### Request A Release

Assuming `my/module` has been initialised and has a compliant `Makefile`, you can prepare a new release like so:

```shell
kaeter -p my/module prepare [ --minor | --major]
```

where without the major or minor switch kaeter increments the third digit(s) of the SemVer version,
second digit(s) with --minor, or the first digit(s) with --major.

It is also possible to prepare multiple modules at the same time by specifying the path parameter (`-p`) multiple times.

### Execute A Release

Assuming the last commit in the repository contains a _release plan_, you may execute said plan with:

```shell
# Without the --really flag a dry run happens
# (ie, all steps except the 'release' one in the Makefile are run)
# With the --nocheckout flag set the commit hash, corresponding to the version of the module,
# will NOT be checked out before releasing
kaeter release --really [--nocheckout]
```

The idea is to have developer run `prepare` and your build hosts run `release` _after_ the release plan has been reviewed and pushed.

## Development & Contributions

Kaeter uses itself for snapshots (not releases), therefore it must have a [CHANGELOG.md](CHANGELOG.md).

gitshell will be replaced by the [git package](./git/) in kaeter itself.
