package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMain(c *check.C) {
	new(Pkglint).Main("pkglint", "-h")

	c.Check(s.Output(), check.Matches, `^\Qusage: pkglint [options] dir...\E\n(?s).+`)
}
