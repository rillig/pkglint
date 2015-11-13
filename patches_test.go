package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklinesPatch_WithComment(c *check.C) {
	lines := mklines("patch-as",
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

	c.Check(G.errors, equals, 0)
	c.Check(G.warnings, equals, 0)
}

func (s *Suite) TestChecklinesPatch_WithoutComment(c *check.C) {
	lines := mklines("patch-as",
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

	c.Check(G.errors, equals, 1) // patch-as:3: Comment expected
	c.Check(G.warnings, equals, 0)
}
