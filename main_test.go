package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMainHelp(c *check.C) {
	new(Pkglint).Main("pkglint", "-h")

	c.Check(s.Output(), check.Matches, `^\Qusage: pkglint [options] dir...\E\n(?s).+`)
}

func (s *Suite) TestMainVersion(c *check.C) {
	new(Pkglint).Main("pkglint", "--version")

	c.Check(s.Output(), check.Matches, `(?:@VERSION@|\d+\.\d+)\n`)
}

func (s *Suite) TestMainNoArgs(c *check.C) {
	defer s.ExpectFatalError(func() {
		c.Check(s.Stderr(), equals, "FATAL: \".\" is not inside a pkgsrc tree.\n")
	})

	new(Pkglint).Main("pkglint")
}
