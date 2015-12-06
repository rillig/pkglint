package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestDetermineUsedVariables_simple(c *check.C) {
	G.mkContext = newMkContext()
	mklines := s.NewMkLines("fname",
		"\t${VAR}")
	mkline := mklines.mklines[0]

	mklines.determineUsedVariables()

	c.Check(len(G.mkContext.varuse), equals, 1)
	c.Check(G.mkContext.varuse["VAR"], equals, mkline)
}

func (s *Suite) TestDetermineUsedVariables_nested(c *check.C) {
	G.mkContext = newMkContext()
	mklines := s.NewMkLines("fname",
		"\t${outer.${inner}}")
	mkline := mklines.mklines[0]

	mklines.determineUsedVariables()

	c.Check(len(G.mkContext.varuse), equals, 3)
	c.Check(G.mkContext.varuse["inner"], equals, mkline)
	c.Check(G.mkContext.varuse["outer."], equals, mkline)
	c.Check(G.mkContext.varuse["outer.*"], equals, mkline)
}

func (s *Suite) TestReShellword(c *check.C) {
	re := `^(?:` + reShellword + `)$`
	matches := check.NotNil
	doesntMatch := check.IsNil

	c.Check(match("", re), doesntMatch)
	c.Check(match("$var", re), matches)
	c.Check(match("$var$var", re), matches)
	c.Check(match("$var;;", re), doesntMatch) // More than one shellword
	c.Check(match("'single-quoted'", re), matches)
	c.Check(match("\"", re), doesntMatch)       // Incomplete string
	c.Check(match("'...'\"...\"", re), matches) // Mixed strings
	c.Check(match("\"...\"", re), matches)
	c.Check(match("`cat file`", re), matches)
}

func (s *Suite) TestResolveVariableRefs_CircularReference(c *check.C) {
	mkline := NewMkLine(NewLine("fname", "1", "GCC_VERSION=${GCC_VERSION}", nil))
	G.pkgContext = newPkgContext(".")
	G.pkgContext.vardef["GCC_VERSION"] = mkline

	resolved := resolveVariableRefs("gcc-${GCC_VERSION}")

	c.Check(resolved, equals, "gcc-${GCC_VERSION}")
}

func (s *Suite) TestResolveVariableRefs_Multilevel(c *check.C) {
	mkline1 := NewMkLine(NewLine("fname", "10", "_=${SECOND}", nil))
	mkline2 := NewMkLine(NewLine("fname", "11", "_=${THIRD}", nil))
	mkline3 := NewMkLine(NewLine("fname", "12", "_=got it", nil))
	G.pkgContext = newPkgContext(".")
	defineVar(mkline1, "FIRST")
	defineVar(mkline2, "SECOND")
	defineVar(mkline3, "THIRD")

	resolved := resolveVariableRefs("you ${FIRST}")

	c.Check(resolved, equals, "you got it")
}

func (s *Suite) TestResolveVariableRefs_SpecialChars(c *check.C) {
	mkline := NewMkLine(NewLine("fname", "dummy", "_=x11", nil))
	G.pkgContext = newPkgContext("category/pkg")
	G.pkgContext.vardef["GST_PLUGINS0.10_TYPE"] = mkline

	resolved := resolveVariableRefs("gst-plugins0.10-${GST_PLUGINS0.10_TYPE}/distinfo")

	c.Check(resolved, equals, "gst-plugins0.10-x11/distinfo")
}

func (s *Suite) TestChecklineRcsid(c *check.C) {
	lines := s.NewLines("fname",
		"$"+"NetBSD: dummy $",
		"$"+"NetBSD$",
		"$"+"Id: dummy $",
		"$"+"Id$",
		"$"+"FreeBSD$")

	for _, line := range lines {
		checklineRcsid(line, ``, "")
	}

	c.Check(s.Output(), equals, ""+
		"ERROR: fname:3: Expected \"$"+"NetBSD$\".\n"+
		"ERROR: fname:4: Expected \"$"+"NetBSD$\".\n"+
		"ERROR: fname:5: Expected \"$"+"NetBSD$\".\n")
}

func (s *Suite) TestMatchVarassign(c *check.C) {
	m, varname, op, value, comment := matchVarassign("C++=c11")

	c.Check(m, equals, true)
	c.Check(varname, equals, "C+")
	c.Check(op, equals, "+=")
	c.Check(value, equals, "c11")
	c.Check(comment, equals, "")
}
