[![Build Status](https://travis-ci.org/rillig/pkglint.svg?branch=master)](https://travis-ci.org/rillig/pkglint)
[![codecov](https://codecov.io/gh/rillig/pkglint/branch/master/graph/badge.svg)](https://codecov.io/gh/rillig/pkglint)

pkglint checks whether a pkgsrc package conforms to the various
conventions established over the years. It produces warnings, errors and
notes and, upon request, explains them.

Before importing a new package or making changes to an existing package,
pkglint should be run in the package's directory to check for common
errors.

See https://www.pkgsrc.org/.

----

For an introduction to programming and extending pkglint,
see [The pkglint tour](codewalk.md).
