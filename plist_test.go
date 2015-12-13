package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklinesPlist(c *check.C) {
	lines := s.NewLines("PLIST",
		"bin/i386/6c",
		"bin/program",
		"@exec ${MKDIR} include/pkgbase",
		"${PLIST.man}man/cat3/strcpy.4",
		"${PLIST.obsolete}@unexec rmdir /tmp")

	checklinesPlist(lines)

	c.Check(s.Output(), equals, ""+
		"ERROR: PLIST:1: Expected \"@comment $"+"NetBSD$\".\n"+
		"WARN: PLIST:1: The bin/ directory should not have subdirectories.\n"+
		"WARN: PLIST:4: Preformatted manual page without unformatted one.\n"+
		"WARN: PLIST:4: Preformatted manual pages should end in \".0\".\n"+
		"WARN: PLIST:5: Please remove this line. It is no longer necessary.\n")
}

func (s *Suite) TestChecklinesPlist_empty(c *check.C) {
	lines := s.NewLines("PLIST",
		"@comment $"+"NetBSD$")

	checklinesPlist(lines)

	c.Check(s.Output(), equals, "WARN: PLIST:1: PLIST files shouldn't be empty.\n")
}

func (s *Suite) TestChecklinesPlist_commonEnd(c *check.C) {
	s.CreateTmpFile(c, "PLIST.common", ""+
		"@comment $"+"NetBSD$\n"+
		"bin/common\n")
	fname := s.CreateTmpFile(c, "PLIST.common_end", ""+
		"@comment $"+"NetBSD$\n"+
		"sbin/common_end\n")

	checklinesPlist(LoadExistingLines(fname, false))

	c.Check(s.OutputCleanTmpdir(), equals, "")
}

func (s *Suite) TestChecklinesPlist_conditional(c *check.C) {
	G.pkg = NewPackage("category/pkgbase")
	G.pkg.plistSubstCond["PLIST.bincmds"] = true
	lines := s.NewLines("PLIST",
		"@comment $"+"NetBSD$",
		"${PLIST.bincmds}bin/subdir/command")

	checklinesPlist(lines)

	c.Check(s.Output(), equals, "WARN: PLIST:2: The bin/ directory should not have subdirectories.\n")
}
