package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestConvertToLogicalLines_nocont(c *check.C) {
	rawText := "" +
		"first line\n" +
		"second line\n"

	lines := convertToLogicalLines("fname_nocont", rawText, false)

	c.Check(lines, check.HasLen, 2)
	c.Check(lines[0].String(), equals, "fname_nocont:1: first line")
	c.Check(lines[1].String(), equals, "fname_nocont:2: second line")
}

func (s *Suite) TestConvertToLogicalLines_cont(c *check.C) {
	rawText := "" +
		"first line \\\n" +
		"second line\n" +
		"third\n"

	lines := convertToLogicalLines("fname_cont", rawText, true)

	c.Check(lines, check.HasLen, 2)
	c.Check(lines[0].String(), equals, "fname_cont:1--2: first line second line")
	c.Check(lines[1].String(), equals, "fname_cont:3: third")
}

func (s *Suite) TestConvertToLogicalLines_contInLastLine(c *check.C) {
	rawText := "" +
		"last line\\"

	lines := convertToLogicalLines("fname_contlast", rawText, true)

	c.Check(lines, check.HasLen, 1)
	c.Check(lines[0].String(), equals, "fname_contlast:1: last line\\")
	c.Check(s.Stdout(), equals, "ERROR: fname_contlast:1: File must end with a newline.\n")
}
