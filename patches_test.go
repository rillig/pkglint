package main

import (
	check "gopkg.in/check.v1"
	"io/ioutil"
	"os"
)

func (s *Suite) TestChecklinesPatch_WithComment(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("patch-WithComment",
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

func (s *Suite) TestChecklinesPatch_WithoutEmptyLine(c *check.C) {
	fname := s.CreateTmpFile(c, "patch-WithoutEmptyLines", "dummy")
	s.UseCommandLine(c, "-Wall", "--autofix")
	lines := s.NewLines(fname,
		"$"+"NetBSD$",
		"Text",
		"--- file.orig",
		"+++ file",
		"@@ -5,3 +5,3 @@",
		" context before",
		"-old line",
		"+old line",
		" context after")

	checklinesPatch(lines)

	c.Check(s.OutputCleanTmpdir(), equals, ""+
		"NOTE: ~/patch-WithoutEmptyLines:2: Empty line expected.\n"+
		"NOTE: ~/patch-WithoutEmptyLines:2: Autofix: inserting a line \"\" before this line.\n"+
		"NOTE: ~/patch-WithoutEmptyLines:3: Empty line expected.\n"+
		"NOTE: ~/patch-WithoutEmptyLines:3: Autofix: inserting a line \"\" before this line.\n"+
		"NOTE: ~/patch-WithoutEmptyLines: Has been auto-fixed. Please re-run pkglint.\n")

	fixed, err := ioutil.ReadFile(fname)
	c.Assert(err, check.IsNil)
	c.Check(string(fixed), equals, ""+
		"$NetBSD$\n"+
		"\n"+
		"Text\n"+
		"\n"+
		"--- file.orig\n"+
		"+++ file\n"+
		"@@ -5,3 +5,3 @@\n"+
		" context before\n"+
		"-old line\n"+
		"+old line\n"+
		" context after\n")
}

func (s *Suite) TestChecklinesPatch_WithoutComment(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("patch-WithoutComment",
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

	c.Check(s.Output(), equals, "ERROR: patch-WithoutComment:3: Each patch must be documented.\n")
}

func (s *Suite) TestChecklineOtherAbsolutePathname(c *check.C) {
	line := NewLine("patch-ag", 1, "+$install -s -c ./bin/rosegarden ${DESTDIR}$BINDIR", nil)

	checklineOtherAbsolutePathname(line, line.text)

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestChecklinesPatch_ErrorCode(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("patch-ErrorCode",
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
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("patch-WrongOrder",
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

	c.Check(s.Output(), equals, "WARN: patch-WrongOrder:7: Unified diff headers should be first ---, then +++.\n")
}

func (s *Suite) TestChecklinesPatch_ContextDiff(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("patch-ctx",
		"$"+"NetBSD$",
		"",
		"diff -cr history.c.orig history.c",
		"*** history.c.orig",
		"--- history.c",
		"***************",
		"*** 11,16 ****",
		"--- 11,24 ----",
		"  context1",
		"  context2",
		"  ",
		"+ addition1",
		"+ ",
		"+ addition3",
		"+ addition4",
		"+ addition5 \"/usr/bin/grep\"",
		"+ ",
		"+ ",
		"+ ",
		"  context4",
		"  context5",
		"  context6",
		"***************",
		"*** 19,24 ****",
		"--- 27,33 ----",
		"  context1",
		"  context2",
		"  ",
		"+ ",
		"  context4",
		"  context5",
		"  context6")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, ""+
		"NOTE: patch-ctx:4: Empty line expected.\n"+
		"WARN: patch-ctx:4: Please use unified diffs (diff -u) for patches.\n"+
		"WARN: patch-ctx:16: Found absolute pathname: /usr/bin/grep\n")
}

func (s *Suite) TestChecklinesPatch_NoPatch(c *check.C) {
	lines := s.NewLines("patch-aa",
		"$"+"NetBSD$",
		"",
		"-- oldfile",
		"++ newfile")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, "ERROR: patch-aa: Contains no patch.\n")
}

func (s *Suite) TestChecklinesPatch_TwoPatches(c *check.C) {
	G.traceOut = os.Stdout
	s.UseCommandLine(c,"-Dtrace")
	lines := s.NewLines("patch-aa",
		"$"+"NetBSD$",
		"",
		"--- oldfile",
		"+++ newfile",
		"@@ -1 +1 @@",
		"-old",
		"+new",
		"--- oldfile2",
		"+++ newfile2",
		"@@ -1 +1 @@",
		"-old",
		"+new")

	checklinesPatch(lines)

	c.Check(s.Output(), equals, ""+
		"ERROR: patch-aa:3: Each patch must be documented.\n"+
		"WARN: patch-aa: Contains patches for 2 files, should be only one.\n")
}
