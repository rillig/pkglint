package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVariableNeedsQuoting(c *check.C) {
	line := NewLine("fname", 1, "dummy", nil)
	G.globalData.InitVartypes()
	pkgnameType := G.globalData.vartypes["PKGNAME"]

	// In Makefile: PKGNAME := ${UNKNOWN}
	vuc := &VarUseContext{pkgnameType, vucTimeParse, vucQuotUnknown, vucExtentUnknown}
	nq := variableNeedsQuoting(line, "UNKNOWN", vuc)

	c.Check(nq, equals, nqDontKnow)
}

func (s *Suite) TestVariableNeedsQuoting_Varbase(c *check.C) {
	line := NewLine("fname", 1, "dummy", nil)
	G.globalData.InitVartypes()

	t1 := getVariableType(line, "FONT_DIRS")

	c.Assert(t1, check.NotNil)
	c.Check(t1.String(), equals, "ShellList of Pathmask")

	t2 := getVariableType(line, "FONT_DIRS.ttf")

	c.Assert(t2, check.NotNil)
	c.Check(t2.String(), equals, "ShellList of Pathmask")
}
