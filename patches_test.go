package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklinesPatch_WithComment(c *check.C) {
	lines := s.NewLines("patch-as",
		"$"+"NetBSD$",
		"",
		"Text",
		"Text",
		"",
		"--- file.orig",
		"+++ file",
		"@@ -5,3 +5,3 @@",
		" context before",
		"-old line",
		"+old line",
		" context after")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestChecklinesPatch_WithoutComment(c *check.C) {
	lines := s.NewLines("patch-as",
		"$"+"NetBSD$",
		"",
		"--- file.orig",
		"+++ file",
		"@@ -5,3 +5,3 @@",
		" context before",
		"-old line",
		"+old line",
		" context after")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, "ERROR: patch-as:3: Comment expected.\n")
}

func (s *Suite) TestChecklineOtherAbsolutePathname(c *check.C) {
	line := NewLine("patch-ag", "1", "+$install -s -c ./bin/rosegarden ${DESTDIR}$BINDIR", nil)

	checklineOtherAbsolutePathname(line, line.text)

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestChecklinesPatch_ErrorCode(c *check.C) {
	lines := s.NewLines("patch-as",
		"$"+"NetBSD$",
		"",
		"*** Error code 1", // Looks like a context diff, but isn’t.
		"",
		"--- file.orig",
		"+++ file",
		"@@ -5,3 +5,3 @@",
		" context before",
		"-old line",
		"+old line",
		" context after")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestChecklinesPatch_WrongOrder(c *check.C) {
	lines := s.NewLines("patch-as",
		"$"+"NetBSD$",
		"",
		"Text",
		"Text",
		"",
		"+++ file",      // Wrong
		"--- file.orig", // Wrong
		"@@ -5,3 +5,3 @@",
		" context before",
		"-old line",
		"+old line",
		" context after")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, "ERROR: patch-as:7: Unified diff headers must be first ---, then +++.\n")
}
