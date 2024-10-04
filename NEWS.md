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
