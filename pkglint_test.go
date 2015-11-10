package main

import (
	"bytes"
	check "gopkg.in/check.v1"
)

func (s *Suite) TestDetermineUsedVariables_simple(c *check.C) {
	G.mkContext = newMkContext()
	line := NewLine("fname", "1", "${VAR}", nil)
	lines := []*Line{line}

	determineUsedVariables(lines)

	c.Assert(len(G.mkContext.varuse), equals, 1)
	c.Assert(G.mkContext.varuse["VAR"], equals, line)
}

func (s *Suite) TestDetermineUsedVariables_nested(c *check.C) {
	G.mkContext = newMkContext()
	line := NewLine("fname", "2", "${outer.${inner}}", nil)
	lines := []*Line{line}

	determineUsedVariables(lines)

	c.Assert(len(G.mkContext.varuse), equals, 3)
	c.Assert(G.mkContext.varuse["inner"], equals, line)
	c.Assert(G.mkContext.varuse["outer."], equals, line)
	c.Assert(G.mkContext.varuse["outer.*"], equals, line)
}

func (s *Suite) TestPrintTable(c *check.C) {
	out := &bytes.Buffer{}

	printTable(out, [][]string{{"hello", "world"}, {"how", "are", "you?"}})

	c.Assert(out.String(), equals, ""+
		"hello  world\n"+
		"how    are    you?\n")
}
