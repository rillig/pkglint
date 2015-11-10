package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestPkgnameFromDistname(c *check.C) {
	G.pkgContext = newPkgContext("dummy")
	G.pkgContext.vardef["PKGNAME"] = NewLine("dummy", "dummy", "dummy", nil)

	c.Check(pkgnameFromDistname("pkgname-1.0", "whatever"), equals, "pkgname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME}", "distname-1.0"), equals, "distname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME:S/dist/pkg/}", "distname-1.0"), equals, "pkgname-1.0")
	c.Check(pkgnameFromDistname("${DISTNAME:S|a|b|g}", "panama-0.13"), equals, "pbnbmb-0.13")
	c.Check(pkgnameFromDistname("${DISTNAME:S|^lib||}", "libncurses"), equals, "ncurses")
	c.Check(pkgnameFromDistname("${DISTNAME:S|^lib||}", "mylib"), equals, "mylib")
}
