package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestConvertToLogicalLines_nocont(c *check.C) {
	phys := []PhysLine{
		{1, "first line\n"},
		{2, "second line\n"},
	}

	lines := convertToLogicalLines("fname", phys, false)

	c.Assert(lines, check.HasLen, 2)
	c.Assert(lines[0].String(), equals, "fname:1: first line")
	c.Assert(lines[1].String(), equals, "fname:2: second line")
}

func (s *Suite) TestConvertToLogicalLines_cont(c *check.C) {
	phys := []PhysLine{
		{1, "first line \\\n"},
		{2, "second line\n"},
		{3, "third\n"},
	}

	lines := convertToLogicalLines("fname", phys, true)

	c.Assert(lines, check.HasLen, 2)
	c.Assert(lines[0].String(), equals, "fname:1--2: first line second line")
	c.Assert(lines[1].String(), equals, "fname:3: third")
}

func (s *Suite) TestConvertToLogicalLines_contInLastLine(c *check.C) {
	physlines := []PhysLine{
		{1, "last line\\"},
	}

	lines := convertToLogicalLines("fname", physlines, true)

	c.Assert(lines, check.HasLen, 1)
	c.Assert(lines[0].String(), equals, "fname:1: last line\\")
}
