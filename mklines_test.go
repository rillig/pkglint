package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMkLines_CheckForUsedComment(c *check.C) {
	s.NewMkLines("Makefile.common",
		"# $"+"NetBSD$",
		"",
		"# used by sysutils/mc",
	).checkForUsedComment("sysutils/mc")

	c.Check(s.Output(), equals, "")

	s.NewMkLines("Makefile.common").checkForUsedComment("category/package")

	c.Check(s.Output(), equals, "")

	s.NewMkLines("Makefile.common",
		"# $"+"NetBSD$",
	).checkForUsedComment("category/package")

	c.Check(s.Output(), equals, "")

	s.NewMkLines("Makefile.common",
		"# $"+"NetBSD$",
		"",
	).checkForUsedComment("category/package")

	c.Check(s.Output(), equals, "")

	s.NewMkLines("Makefile.common",
		"# $"+"NetBSD$",
		"",
		"VARNAME=\tvalue",
	).checkForUsedComment("category/package")

	c.Check(s.Output(), equals, "WARN: Makefile.common:2: Please add a line \"# used by category/package\" here.\n")

	s.NewMkLines("Makefile.common",
		"# $"+"NetBSD$",
		"#",
		"#",
	).checkForUsedComment("category/package")

	c.Check(s.Output(), equals, "WARN: Makefile.common:3: Please add a line \"# used by category/package\" here.\n")
}
