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

	c.Assert("fname", equals, lines[0].fname)
	c.Assert("1", equals, lines[0].lines)
	c.Assert("first line", equals, lines[0].text)
	c.Assert("fname", equals, lines[1].fname)
	c.Assert("2", equals, lines[1].lines)
	c.Assert("second line", equals, lines[1].text)
}

func (s *Suite) TestConvertToLogicalLines_cont(c *check.C) {
	phys := []PhysLine{
		{1, "first line \\\n"},
		{2, "second line\n"},
		{3, "third\n"},
	}

	lines := convertToLogicalLines("fname", phys, true)

	c.Assert(len(lines), equals, 2)
	c.Assert(lines[0].lines, equals, "1--2")
	c.Assert("first line second line", equals, lines[0].text)
	c.Assert("3", equals, lines[1].lines)
	c.Assert("third", equals, lines[1].text)
}

func (s *Suite) TestConvertToLogicalLines_contInLastLine(c *check.C) {
	physlines := []PhysLine{
		{1, "last line\\"},
	}

	lines := convertToLogicalLines("fname", physlines, true)

	c.Assert(lines[0].fname, equals, "fname")
	c.Assert(lines[0].lines, equals, "1")
	c.Assert(lines[0].text, equals, "last line\\")
}
