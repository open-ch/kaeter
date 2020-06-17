# kætər-police

The purpose of this tool is to enforce best practices for the development and release of packages.
Among them:
- no package that has to be released should lack a README.md or a CHANGELOG.md file.
- any release should be tracked in the changelog.

## HowTo

```
# Checks for compliance of all modules (ie, any directory that has a `versions.yml` file.
kaeter-police -p <path_to_repo> check
```
