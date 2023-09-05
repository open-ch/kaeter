# native git wrapper

Not go-git, not [gitshell](https://github.com/open-ch/go-libs/tree/master/gitshell).

New functionality will be added here, eventually gitshell functionality will be
reimplemented here as well.

The goal of this package is to allow:
1. Making git calls
2. Keeping the syntax as close to the git syntax as possible
3. Avoiding hidden information (flags automatically passed, ...)
4. Adding extra flags to a command easily

## Why not go-git?

Historically it was implemented (gitshell) relying on the local git install, perhaps
using quirks of git versions at the time, or perhaps relying on lower level feature
go-git did not provide at the time of implementation. This might no longer hold
true today.

Also by design gitshell did not require instances or objects, functions could be called
without needing to first create an instance or pass an instance around, acting more
like a singleton. This still holds true, a single line call is more readable than
instantiating and object, preparing an action and then executing it over multiple lines.

## Why not gitshell?

Gitshell doesn't have state, it's not a singleton, it's only an interface to git. This requires
passing the working director (the repository path) everytime. It keeps the functions somewhat
more pure but requires repetitive dangling parameters in the front. Using a singleton approach
would allow setting the working path once, since kaeter works on a single repository this will
make it more DRY.

Gitshell doesn't allow expanding parameters. It's nicely and strongly typed but to pass one
more flag to a git command requires updating the library.
