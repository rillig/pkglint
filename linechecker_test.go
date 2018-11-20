package main

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_LineChecker_CheckAbsolutePathname(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("Makefile", 1, "# dummy")

	ck := LineChecker{line}
	ck.CheckAbsolutePathname("bindir=/bin")
	ck.CheckAbsolutePathname("bindir=/../lib")
	ck.CheckAbsolutePathname("cat /dev/null")
	ck.CheckAbsolutePathname("cat /dev/tty")
	ck.CheckAbsolutePathname("cat /dev/zero")
	ck.CheckAbsolutePathname("cat /dev/stdin")
	ck.CheckAbsolutePathname("cat /dev/stdout")
	ck.CheckAbsolutePathname("cat /dev/stderr")
	ck.CheckAbsolutePathname("printf '#! /bin/sh\\nexit 0'")

	// This is not a filename at all, but certainly looks like one.
	// Nevertheless, pkglint doesn't fall into the trap.
	ck.CheckAbsolutePathname("sed -e /usr/s/usr/var/g")

	t.CheckOutputLines(
		"WARN: Makefile:1: Found absolute pathname: /bin",
		"WARN: Makefile:1: Found absolute pathname: /dev/stdin",
		"WARN: Makefile:1: Found absolute pathname: /dev/stdout",
		"WARN: Makefile:1: Found absolute pathname: /dev/stderr")
}

// It is unclear whether pkglint should check for absolute pathnames by default.
// It might be useful, but all the code surrounding this check was added for
// theoretical reasons instead of a practical bug. Therefore the code is still
// there, it is just not enabled by default.
func (s *Suite) Test_LineChecker_CheckAbsolutePathname__disabled_by_default(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine( /* none, which means -Wall is suppressed */ )
	line := t.NewLine("Makefile", 1, "# dummy")

	LineChecker{line}.CheckAbsolutePathname("bindir=/bin")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_LineChecker_CheckTrailingWhitespace(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("Makefile", 32, "The line must go on   ")

	LineChecker{line}.CheckTrailingWhitespace()

	t.CheckOutputLines(
		"NOTE: Makefile:32: Trailing whitespace.")
}
