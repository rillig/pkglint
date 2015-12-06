package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestPkgnameFromDistname(c *check.C) {
	G.pkgContext = newPkgContext("dummy")
	G.pkgContext.vardef["PKGNAME"] = NewMkLine(NewLine("dummy", "dummy", "dummy", nil))

	c.Check(pkgnameFromDistname("pkgname-1.0", "whatever"), equals, "pkgname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME}", "distname-1.0"), equals, "distname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME:S/dist/pkg/}", "distname-1.0"), equals, "pkgname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME:S|a|b|g}", "panama-0.13"), equals, "pbnbmb-0.13")
	c.Check(pkgnameFromDistname("${DISTNAME:S|^lib||}", "libncurses"), equals, "ncurses")
	c.Check(pkgnameFromDistname("${DISTNAME:S|^lib||}", "mylib"), equals, "mylib")
}

func (s *Suite) TestChecklinesPackageMakefileVarorder(c *check.C) {
	s.UseCommandLine(c, "-Worder")
	G.pkgContext = newPkgContext("x11/9term")

	ChecklinesPackageMakefileVarorder(s.NewMkLines("Makefile",
		"# $"+"NetBSD$",
		"",
		"DISTNAME=9term",
		"CATEGORIES=x11"))

	c.Check(s.Output(), equals, "")

	ChecklinesPackageMakefileVarorder(s.NewMkLines("Makefile",
		"# $"+"NetBSD$",
		"",
		"DISTNAME=9term",
		"CATEGORIES=x11",
		"",
		".include \"../../mk/bsd.pkg.mk\""))

	c.Check(s.Output(), equals, ""+
		"WARN: Makefile:6: COMMENT should be set here.\n"+
		"WARN: Makefile:6: LICENSE should be set here.\n")
}

func (s *Suite) TestGetNbpart(c *check.C) {
	G.pkgContext = newPkgContext("category/pkgbase")
	G.pkgContext.vardef["PKGREVISION"] = NewMkLine(NewLine("Makefile", "1", "PKGREVISION=14", nil))

	c.Check(getNbpart(), equals, "nb14")

	G.pkgContext.vardef["PKGREVISION"] = NewMkLine(NewLine("Makefile", "1", "PKGREVISION=asdf", nil))

	c.Check(getNbpart(), equals, "")
}
