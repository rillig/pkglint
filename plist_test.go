package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklinesPlist(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	lines := s.NewLines("PLIST",
		"bin/i386/6c",
		"bin/program",
		"etc/my.cnf",
		"etc/rc.d/service",
		"@exec ${MKDIR} include/pkgbase",
		"info/dir",
		"lib/libc.so.6",
		"lib/libc.la",
		"${PLIST.man}man/cat3/strcpy.4",
		"${PLIST.obsolete}@unexec rmdir /tmp",
		"share/tzinfo",
		"share/tzinfo")

	checklinesPlist(lines)

	c.Check(s.Output(), equals, ""+
		"ERROR: PLIST:1: Expected \"@comment $NetBSD$\".\n"+
		"WARN: PLIST:1: The bin/ directory should not have subdirectories.\n"+
		"WARN: PLIST:2: Manual page missing for bin/program.\n"+
		"ERROR: PLIST:3: Configuration files must not be registered in the PLIST. Please use the CONF_FILES framework, which is described in mk/pkginstall/bsd.pkginstall.mk.\n"+
		"ERROR: PLIST:4: RCD_SCRIPTS must not be registered in the PLIST. Please use the RCD_SCRIPTS framework.\n"+
		"ERROR: PLIST:6: \"info/dir\" must not be listed. Use install-info to add/remove an entry.\n"+
		"WARN: PLIST:7: Redundant library found. The libtool library is in line 8.\n"+
		"WARN: PLIST:8: \"lib/libc.la\" should be sorted before \"lib/libc.so.6\".\n"+
		"WARN: PLIST:9: Preformatted manual page without unformatted one.\n"+
		"WARN: PLIST:9: Preformatted manual pages should end in \".0\".\n"+
		"WARN: PLIST:10: Please remove this line. It is no longer necessary.\n"+
		"ERROR: PLIST:12: Duplicate filename \"share/tzinfo\", already appeared in PLIST:11.\n")
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

func (s *Suite) TestChecklinesPlist_sorting(c *check.C) {
	s.UseCommandLine(c, "-Wplist-sort")
	lines := s.NewLines("PLIST",
		"@comment $"+"NetBSD$",
		"sbin/i386/6c",
		"sbin/program",
		"bin/otherprogram",
		"${PLIST.conditional}bin/cat")

	checklinesPlist(lines)

	c.Check(s.Output(), equals, ""+
		"WARN: PLIST:4: \"bin/otherprogram\" should be sorted before \"sbin/program\".\n"+
		"WARN: PLIST:5: \"bin/cat\" should be sorted before \"bin/otherprogram\".\n")
}
