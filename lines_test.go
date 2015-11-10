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

	c.Check(lines, check.HasLen, 2)
	c.Check(lines[0].String(), equals, "fname:1: first line")
	c.Check(lines[1].String(), equals, "fname:2: second line")
}

func (s *Suite) TestConvertToLogicalLines_cont(c *check.C) {
	phys := []PhysLine{
		{1, "first line \\\n"},
		{2, "second line\n"},
		{3, "third\n"},
	}

	lines := convertToLogicalLines("fname", phys, true)

	c.Check(lines, check.HasLen, 2)
	c.Check(lines[0].String(), equals, "fname:1--2: first line second line")
	c.Check(lines[1].String(), equals, "fname:3: third")
}

func (s *Suite) TestConvertToLogicalLines_contInLastLine(c *check.C) {
	physlines := []PhysLine{
		{1, "last line\\"},
	}

	lines := convertToLogicalLines("fname", physlines, true)

	c.Check(lines, check.HasLen, 1)
	c.Check(lines[0].String(), equals, "fname:1: last line\\")
}
