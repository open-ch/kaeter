# Kætər Tooling


This module holds the `kaeter` cli and some assorted tooling like `kaeter-police`.

Things are still a bit rough, so your mileage may vary.

## Purpose

The `kaeter` CLI seeks to enable _descriptive releases_ and to provide an answer to this:

> How do I release this and what is the next version number?

While striving to do so in a way that integrates nicely with any CI:
- it allows to developpers to _request_ the release of something, and then to have the CI do the actual release.
- it does not require the CI to push anything to the repo, like an updated version or a release commit

The tool is aimed at fat repositories with multiple _deliverables_ living side by side, but is also regularly used on small repos.

`kaeter-police` is a separate CLI that lets you automate some sanity checks around `kaeter` modules: existence of a readme,
as well as documented releases in a changelog.

See the respective submodule READMEs for [kaeter](kaeter) and [kaeter-police](kaeter-police) for more details.

## Installation

We're sorry to require a few manual steps at the moment, this may be improved upon in the future.

```
# Clone this repo to somewhere
git clone https://github.com/open-ch/kaeter
cd kaeter

# Install all submodules
go install ./...

# The binaries should be in $GOPATH/bin
ls `go env GOPATH`/bin
```

# Submodules

## `kaeter`
`kaeter` provides a generic way to version and release software modules or _deliverables_, be they docker images,
go binaries, jars or whatnot.

The genericity is obtained by relying on a `Makefile` declaring four targets, to which a `VERSION` environment variable
will be passed, which will contain the current version being built:

- `build`
- `test`
- `snapshot`
- `release`

Look at the `kaeter` submodule for mor details.

## `kaeter-police`

This tool enforces a few things around `kaeter` modules, namely:

- every module (ie, anything that has a `versions.yml` file) needs a `README.md` and a `CHANGELOG.md` files
- every released version needs an entry in the `CHANGELG.md` file


## Additional Notes

We're sorry that the tests published to github don't work yet, as there is some difference
between the internal and the public structure.

## License

Please see [LICENSE](LICENSE).
