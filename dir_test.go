package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestCheckDirent(c *check.C) {
	s.CreateTmpFile(c, "empty", "")

	CheckDirent(s.tmpdir)

	c.Check(s.OutputCleanTmpdir(), equals, "ERROR: ~: Cannot determine the pkgsrc root directory for \"~\".\n")
}
