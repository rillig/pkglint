package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestCheckdirToplevel(c *check.C) {
	s.CreateTmpFile(c, "Makefile",
		"# $"+"NetBSD$",
		"",
		"SUBDIR+= x11",
		"SUBDIR+=\tarchivers",
		"SUBDIR+=\tccc",
		"SUBDIR+=\tccc",
		"#SUBDIR+=\tignoreme",
		"SUBDIR+=\tnonexisting", // This just doesn’t happen in practice.
		"SUBDIR+=\tbbb")
	s.CreateTmpFile(c, "archivers/Makefile", "")
	s.CreateTmpFile(c, "bbb/Makefile", "")
	s.CreateTmpFile(c, "ccc/Makefile", "")
	s.CreateTmpFile(c, "x11/Makefile", "")
	G.globalData.InitVartypes()
	G.currentDir = s.tmpdir

	checkdirToplevel()

	c.Check(s.Output(), equals, ""+
		"WARN: "+s.tmpdir+"/Makefile:3: Indentation should be a single tab character.\n" +
		"ERROR: "+s.tmpdir+"/Makefile:6: Each subdir must only appear once.\n" +
		"WARN: "+s.tmpdir+"/Makefile:7: \"ignoreme\" commented out without giving a reason.\n"+
		"WARN: "+s.tmpdir+"/Makefile:9: bbb should come before ccc\n")
}
