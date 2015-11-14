package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklineMkVartype_SimpleType(c *check.C) {
	G.opts.optWarnTypes = true
	G.opts.optDebugUnchecked = true
	line := NewLine("fname", "1", "dummy", nil)
	initacls()

	vartype1 := G.globalData.getVartypes()["COMMENT"]
	c.Assert(vartype1, check.NotNil)
	c.Check(vartype1.guessed, equals, NOT_GUESSED)

	vartype := getVariableType(line, "COMMENT")

	c.Assert(vartype, check.NotNil)
	c.Check(vartype.basicType, equals, "Comment")
	c.Check(vartype.guessed, equals, NOT_GUESSED)
	c.Check(vartype.kindOfList, equals, LK_NONE)

	checklineMkVartype(line, "COMMENT", "=", "A nice package", "")

	c.Check(s.Stdout(), equals, "WARN: fname:1: COMMENT should not begin with \"A\".\n")
}

func (s *Suite) TestChecklineMkVartype(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)
	initacls()

	checklineMkVartype(line, "DISTNAME", "=", "gcc-${GCC_VERSION}", "")
}
