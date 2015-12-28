package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMkLines_AutofixConditionalIndentation(c *check.C) {
	s.UseCommandLine(c, "--autofix", "-Wspace")
	tmpfile := s.CreateTmpFile(c, "fname.mk", "")
	mklines := s.NewMkLines(tmpfile,
		"# $"+"NetBSD$",
		".if defined(A)",
		".for a in ${A}",
		".if defined(C)",
		".endif",
		".endfor",
		".endif")

	mklines.check()

	c.Check(s.OutputCleanTmpdir(), equals, ""+
		"NOTE: ~/fname.mk:3: Autofix: replacing \".\" with \".  \".\n"+
		"NOTE: ~/fname.mk:4: Autofix: replacing \".\" with \".    \".\n"+
		"NOTE: ~/fname.mk:5: Autofix: replacing \".\" with \".    \".\n"+
		"NOTE: ~/fname.mk:6: Autofix: replacing \".\" with \".  \".\n"+
		"NOTE: ~/fname.mk: Has been auto-fixed. Please re-run pkglint.\n")
	c.Check(s.LoadTmpFile(c, "fname.mk"), equals, ""+
		"# $NetBSD$\n"+
		".if defined(A)\n"+
		".  for a in ${A}\n"+
		".    if defined(C)\n"+
		".    endif\n"+
		".  endfor\n"+
		".endif\n")
}
