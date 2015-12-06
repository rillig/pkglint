package main

import (
	check "gopkg.in/check.v1"
)

// In variable assignments, a plain '#' introduces a line comment, unless
// it is escaped by a backslash. In shell commands, on the other hand, it
// is interpreted literally.
func (s *Suite) TestParselineMk_Varassign(c *check.C) {
	line := NewMkLine(NewLine("fname", 1, "SED_CMD=\t's,\\#,hash,g'", nil))

	c.Check(line.Varname(), equals, "SED_CMD")
	c.Check(line.Value(), equals, "'s,#,hash,g'")
}

func (s *Suite) TestParselineMk_Shellcmd(c *check.C) {
	line := NewMkLine(NewLine("fname", 1, "\tsed -e 's,\\#,hash,g'", nil))

	c.Check(line.Shellcmd(), equals, "sed -e 's,\\#,hash,g'")
}
