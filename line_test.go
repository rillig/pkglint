package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestLineModify(c *check.C) {
	s.UseCommandLine(c, "--show-autofix")

	line := NewLine("fname", 1, "dummy", s.NewRawLines(1, "original\n"))

	c.Check(line.changed, equals, false)
	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(1, "original\n"))

	line.autofixReplaceRegexp(`(.)(.*)(.)`, "$3$2$1")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(1, "original\n", "lriginao\n"))

	line.changed = false
	line.autofixReplace("i", "u")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(1, "original\n", "lruginao\n"))
	c.Check(line.raw[0].textnl, equals, "lruginao\n")

	line.changed = false
	line.autofixReplace("lruginao", "middle")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(1, "original\n", "middle\n"))
	c.Check(line.raw[0].textnl, equals, "middle\n")

	line.autofixInsertBefore("before")
	line.autofixInsertBefore("between before and middle")
	line.autofixInsertAfter("between middle and after")
	line.autofixInsertAfter("after")

	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(
		0, "", "before\n",
		0, "", "between before and middle\n",
		1, "original\n", "middle\n",
		0, "", "between middle and after\n",
		0, "", "after\n"))

	line.autofixDelete()

	c.Check(line.rawLines(), check.DeepEquals, s.NewRawLines(
		0, "", "before\n",
		0, "", "between before and middle\n",
		1, "original\n", "",
		0, "", "between middle and after\n",
		0, "", "after\n"))
}

func (s *Suite) TestLine_CheckAbsolutePathname(c *check.C) {
	line := NewLine("Makefile", 1, "# dummy", nil)

	line.checkAbsolutePathname("bindir=/bin")
	line.checkAbsolutePathname("bindir=/../lib")

	c.Check(s.Output(), equals, "WARN: Makefile:1: Found absolute pathname: /bin\n")
}

func (s *Suite) TestShowAutofix_replace(c *check.C) {
	s.UseCommandLine(c, "--show-autofix", "--source")
	line := NewLineMulti("Makefile", 27, 29, "# old", s.NewRawLines(
		27, "before\n",
		28, "The old song\n",
		29, "after\n"))

	if !line.autofixReplace("old", "new") {
		line.warn0("Using \"old\" is deprecated.")
	}

	c.Check(s.Output(), equals, ""+
		"\n"+
		"> before\n"+
		"- The old song\n"+
		"+ The new song\n"+
		"> after\n"+
		"WARN: Makefile:27--29: Using \"old\" is deprecated.\n"+
		"AUTOFIX: Makefile:27--29: Replacing \"old\" with \"new\".\n")
}

func (s *Suite) TestShowAutofix_insert(c *check.C) {
	s.UseCommandLine(c, "--show-autofix", "--source")
	line := NewLine("Makefile", 30, "original", s.NewRawLines(30, "original\n"))

	if !line.autofixInsertBefore("inserted") {
		line.warn0("Dummy")
	}

	c.Check(s.Output(), equals, ""+
		"\n"+
		"+ inserted\n"+
		"> original\n"+
		"WARN: Makefile:30: Dummy\n"+
		"AUTOFIX: Makefile:30: Inserting a line \"inserted\" before this line.\n")
}

func (s *Suite) TestShowAutofix_delete(c *check.C) {
	s.UseCommandLine(c, "--show-autofix", "--source")
	line := NewLine("Makefile", 30, "to be deleted", s.NewRawLines(30, "to be deleted\n"))

	if !line.autofixDelete() {
		line.warn0("Dummy")
	}

	c.Check(s.Output(), equals, ""+
		"\n"+
		"- to be deleted\n"+
		"WARN: Makefile:30: Dummy\n"+
		"AUTOFIX: Makefile:30: Deleting this line.\n")
}
