package main

import (
	check "gopkg.in/check.v1"
)

// In variable assignments, a plain '#' introduces a line comment, unless
// it is escaped by a backslash. In shell commands, on the other hand, it
// is interpreted literally.
func (s *Suite) TestParselineMk_VarAssign(c *check.C) {
	line := NewLine("fname", "1", "SED_CMD=\t's,\\#,hash,g'", nil)

	parselineMk(line)

	c.Check(line.extra["varname"], equals, "SED_CMD")
	c.Check(line.extra["value"], equals, "'s,#,hash,g'")
}
