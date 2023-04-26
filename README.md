# Kætər Tooling

This module holds the `kaeter` cli mono-repo tool.

Things are still a bit rough, so your mileage may vary.

## Purpose

The `kaeter` CLI seeks to enable _descriptive releases_ and to provide an answer to this:

> How do I release this and what is the next version number?

While striving to do so in a way that integrates nicely with any CI:
- it allows to developpers to _request_ the release of something, and then to have the CI do the actual release,
- it does not require the CI to push anything to the repo, like an updated version or a release commit.

The tool is aimed at fat repositories with multiple _deliverables_ living side by side, but is also regularly used on small repos.

## Installation

```
# Clone this repo to somewhere
git clone https://github.com/open-ch/kaeter
cd kaeter

# Install all submodules
go mod tidy
go install ./...

# The binaries should be in $GOPATH/bin
ls `go env GOPATH`/bin
```

# Commands and Subcommands

## `kaeter`
`kaeter` provides a generic way to version and release software modules or _deliverables_, be they docker images,
go binaries, jars or whatnot.

The genericity is obtained by relying on a `Makefile` declaring four targets, to which a `VERSION` environment variable
will be passed, which will contain the current version being built:

- `build`
- `test`
- `release`
- `snapshot` (optional)

Look at the `kaeter --help` for mor details.

### `kaeter lint`

This command enforces a few things around `kaeter` modules, namely:

- every module (ie, anything that has a `versions.yaml` file) needs `README.md` and `CHANGELOG.md` files,
- every released version needs an entry in `CHANGELG.md` file.

```
# Checks for compliance of all modules (ie, any directory that has a `versions.yml` file).
kaeter lint --path <path_to_repo>
```

## Additional Notes

We're sorry that the tests published to github don't work yet, as there is some difference
between the internal and the public structure.

## License

Please see [LICENSE](LICENSE).
