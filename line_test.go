package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestLineModify(c *check.C) {
	line := NewLine("fname", 1, "dummy", []*RawLine{{1, "original\n"}})

	c.Check(line.changed, equals, false)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "original\n"}})

	line.autofixReplaceRegexp(`(.)(.*)(.)`, "$3$2$1")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "lriginao\n"}})

	line.changed = false
	line.autofixReplace("i", "u")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "lruginao\n"}})
	c.Check(line.raw[0].textnl, equals, "lruginao\n")

	line.changed = false
	line.autofixReplace("lruginao", "middle")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "middle\n"}})
	c.Check(line.raw[0].textnl, equals, "middle\n")

	line.autofixInsertBefore("before")
	line.autofixInsertBefore("between before and middle")
	line.autofixInsertAfter("between middle and after")
	line.autofixInsertAfter("after")

	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{
		{0, "before\n"},
		{0, "between before and middle\n"},
		{1, "middle\n"},
		{0, "between middle and after\n"},
		{0, "after\n"}})

	line.autofixDelete()

	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{
		{0, "before\n"},
		{0, "between before and middle\n"},
		{0, "between middle and after\n"},
		{0, "after\n"}})
}

func (s *Suite) TestLine_CheckAbsolutePathname(c *check.C) {
	line := NewLine("Makefile", 1, "# dummy", nil)

	line.checkAbsolutePathname("bindir=/bin")
	line.checkAbsolutePathname("bindir=/../lib")

	c.Check(s.Output(), equals, "WARN: Makefile:1: Found absolute pathname: /bin\n")
}
