package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestParseMkCond_NotEmptyMatch(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)

	cond := parseMkCond(line, "!empty(USE_LIBTOOL:M[Yy][Ee][Ss])")

	c.Check(cond, check.DeepEquals, NewTree("not", NewTree("empty", NewTree("match", "USE_LIBTOOL", "[Yy][Ee][Ss]"))))
}

func (s *Suite) TestParseMkCond_Compare(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)

	cond := parseMkCond(line, "${VARNAME} != \"Value\"")

	c.Check(cond, check.DeepEquals, NewTree("!=", NewTree("var", "VARNAME"), NewTree("string", "Value")))
}
