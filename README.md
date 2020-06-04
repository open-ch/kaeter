# Kætər Tooling

This module holds the `kaeter` cli and some assorted tooling like `kaeter-police`.

Things are still a bit rough, so your mileage may vary.

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
