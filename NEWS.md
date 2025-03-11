# 23.15.3 (2025-03-11)

Fix a wrong note about WRKSRC being redundantly defined when the definition
was in fact necessary. This only affects packages that define GITHUB_TAG.

# 23.15.2 (2025-03-05)

Fix the wrong error message about a package not including its own options.mk
when that file is included via a more complicated path, for example, from
Makefile.common.

# 23.15.1 (2025-02-26)

Remove leftover tests that failed after removing the undocumented profiling
option.

Still require PATCHDIR to be well-formed; the directory it points to just
doesn't have to exist.

# 23.15.0 (2025-02-26)

Complain if a package has an options.mk file but doesn't included it.

Don't complain about pointing PATCHDIR to a nonexistent directory
as long as PATCHDIR and DISTINFO_FILE match.

# 23.14.0 (2025-02-17)

Fix wrong warnings about MASTER_SITE_BACKUP being not known.

Reduce the amount of log data when tracing variables that repeatedly
use `+=` to append to a variable.

Support the `.-include` directive, the `.ifmake` directive and the
built-in `<` variable, to be able to check standalone makefiles
outside pkgsrc.

Complain about merge conflicts in makefiles, which were silently ignored
before.

Complain about package patterns whose bounds contradict each other,
such as `pkgbase>=2<1`, as these never match and are probably typos.

Do not warn about pkgsrc-wip packages that are missing a COMMIT_MSG file,
as long as they have a TODO file, as it is fine to have work-in-progress
packages in pkgsrc-wip.

# 23.13.0 (2025-01-28)

Check language version variables for Lua, PHP, Python and Ruby.

# 23.12.0 (2025-01-27)

Allow checking doc/pkg-vulnerabilities for malformed package patterns.

In simple package makefiles, check for the order of common package variables.
Be more specific about what to fix, and apply the check to more packages than
before.

# 23.11.0 (2025-01-21)

Don't complain about ignored files in patches/.

Don't check for deprecated variables outside pkgsrc.

Warn about unusual single-character variables.

Fix parsing of the ':!cmd!', ':S' and ':C' modifiers.

Don't simplify unsatisfiable 'empty(VAR:M)' condition.

Fix contradictory warnings about GMAKE_REQD, which is not a list,
even though it follows the naming convention for list variables.

Treat hyphens in package versions as errors, as the full package name
"pkgbase-3g-7.4" is ambiguous, as the version could either start with
"3g" or with "7", and depending on the context, either of these is
used.

# 23.10.0 (2024-12-14)

Check dependency patterns that include alternatives enclosed in braces,
such as {ssh,openssh}>=0.

Fix wrong warnings about invalid dependency patterns,
include helpful details, turn the warnings into errors.

Fix panic when guessing the type of a tool variable.

# 23.9.1 (2024-12-07)

Fix leftover bmake placeholder in test.

# 23.9.0 (2024-12-07)

Warn about `DISTINFO_FILE` and `PATCHDIR` that don't correspond.

# 23.8.1 (2024-12-02)

Fix wrong warning and autofix involving _ULIMIT_CMD, which was detected
as an "unknown shell command".

Reduce punctuation in the debug log when tracing function calls.

# 23.8.0 (2024-10-04)

Prohibit vertical bar in COMMENT, to avoid generating syntactically wrong
INDEX files.

Explain in which cases a `distinfo` file is not needed.

Fix detection of redundant trailing semicolon at the end of a command line.
A semicolon that is escaped is not redundant, this pattern is often found
in `find` commands.

Recognize indirect modifiers such as `${VAR:${M_indirect}}`.

# 23.7.0 (2024-09-12)

Allow the '::=' modifier family in the pkgsrc infrastructure.

Mark USE_CMAKE as deprecated.

Fix the -Wall option not to imply -Werror.

Note redundant trailing semicolons in shell commands.

# 23.6.0 (2024-07-24)

Ignore all Git and GitHub files.

# 23.5.0 (2024-07-16)

Added the -Werror option, which treats warnings as errors.

# 23.4.2 (2024-05-07)

Fixed the wrong suggestion that list variables or variables of an unknown type
could be compared using `${LIST} == word` instead of `${LIST:Mword}`.

Fixed the wrong warning when a value is appended using `+=` to a variable of
unknown type.
