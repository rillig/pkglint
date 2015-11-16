package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklineMkVartype_SimpleType(c *check.C) {
	s.UseCommandLine("-Wtypes", "-Dunchecked")
	G.globalData.InitVartypes()
	line := NewLine("fname", "1", "dummy", nil)

	vartype1 := G.globalData.vartypes["COMMENT"]
	c.Assert(vartype1, check.NotNil)
	c.Check(vartype1.guessed, equals, NOT_GUESSED)

	vartype := getVariableType(line, "COMMENT")

	c.Assert(vartype, check.NotNil)
	c.Check(vartype.checker.name, equals, "Comment")
	c.Check(vartype.guessed, equals, NOT_GUESSED)
	c.Check(vartype.kindOfList, equals, LK_NONE)

	checklineMkVartype(line, "COMMENT", "=", "A nice package", "")

	c.Check(s.Stdout(), equals, "WARN: fname:1: COMMENT should not begin with \"A\".\n")
}

func (s *Suite) TestChecklineMkVartype(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)
	G.globalData.InitVartypes()

	checklineMkVartype(line, "DISTNAME", "=", "gcc-${GCC_VERSION}", "")
}

func (s *Suite) TestChecklineMkVaralign(c *check.C) {
	s.UseCommandLine("-Wspace")
	lines := s.NewLines("file.mk",
		"VAR=   value",    // Indentation 7, is not fixed.
		"VAR=    value",   // Indentation 8, is fixed.
		"VAR= \tvalue",    // Mixed indentation 8, is fixed.
		"VAR=   \tvalue",  // Mixed indentation 8, is fixed.
		"VAR=    \tvalue", // Mixed indentation 16, is fixed.
		"VAR=\tvalue")     // Already aligned with tabs, left unchanged.

	for _, line := range lines {
		ChecklineMkVaralign(line)
	}

	c.Check(lines[0].changed, equals, false)
	c.Check(lines[0].rawLines()[0].String(), equals, "1:VAR=   value\n")
	c.Check(lines[1].changed, equals, true)
	c.Check(lines[1].rawLines()[0].String(), equals, "2:VAR=\tvalue\n")
	c.Check(lines[2].changed, equals, true)
	c.Check(lines[2].rawLines()[0].String(), equals, "3:VAR=\tvalue\n")
	c.Check(lines[3].changed, equals, true)
	c.Check(lines[3].rawLines()[0].String(), equals, "4:VAR=\tvalue\n")
	c.Check(lines[4].changed, equals, true)
	c.Check(lines[4].rawLines()[0].String(), equals, "5:VAR=\t\tvalue\n")
	c.Check(lines[5].changed, equals, false)
	c.Check(lines[5].rawLines()[0].String(), equals, "6:VAR=\tvalue\n")
	c.Check(s.Output(), equals, ""+
		"NOTE: file.mk:1: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:2: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:3: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:4: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:5: Alignment of variable values should be done with tabs, not spaces.\n")
	c.Check(tabLength("VAR=    \t"), equals, 16)
}
