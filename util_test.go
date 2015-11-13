package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMkopSubst_middle(c *check.C) {
	c.Check(mkopSubst("pkgname", false, "kgna", false, "ri", false), equals, "prime")
	c.Check(mkopSubst("pkgname", false, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_left(c *check.C) {
	c.Check(mkopSubst("pkgname", true, "kgna", false, "ri", false), equals, "pkgname")
	c.Check(mkopSubst("pkgname", true, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_right(c *check.C) {
	c.Check(mkopSubst("pkgname", false, "kgna", true, "ri", false), equals, "pkgname")
	c.Check(mkopSubst("pkgname", false, "pkgname", true, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_leftRight(c *check.C) {
	c.Check(mkopSubst("pkgname", true, "kgna", true, "ri", false), equals, "pkgname")
	c.Check(mkopSubst("pkgname", false, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_all(c *check.C) {
	c.Check(mkopSubst("aaaaa", false, "a", false, "b", true), equals, "bbbbb")
	c.Check(mkopSubst("aaaaa", true, "a", false, "b", true), equals, "baaaa")
	c.Check(mkopSubst("aaaaa", false, "a", true, "b", true), equals, "aaaab")
	c.Check(mkopSubst("aaaaa", true, "a", true, "b", true), equals, "aaaaa")
}

func (s *Suite) TestReplaceFirst(c *check.C) {
	m, rest := replaceFirst("a+b+c+d", `(\w)(.)(\w)`, "X")

	c.Assert(m, check.NotNil)
	c.Check(m, check.DeepEquals, []string{"a+b", "a", "+", "b"})
	c.Check(rest, equals, "X+c+d")
}
