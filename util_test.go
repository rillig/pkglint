package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestMkopSubst_middle(c *check.C) {
	c.Assert(mkopSubst("pkgname", false, "kgna", false, "ri", false), equals, "prime")
	c.Assert(mkopSubst("pkgname", false, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_left(c *check.C) {
	c.Assert(mkopSubst("pkgname", true, "kgna", false, "ri", false), equals, "pkgname")
	c.Assert(mkopSubst("pkgname", true, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_right(c *check.C) {
	c.Assert(mkopSubst("pkgname", false, "kgna", true, "ri", false), equals, "pkgname")
	c.Assert(mkopSubst("pkgname", false, "pkgname", true, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_leftRight(c *check.C) {
	c.Assert(mkopSubst("pkgname", true, "kgna", true, "ri", false), equals, "pkgname")
	c.Assert(mkopSubst("pkgname", false, "pkgname", false, "replacement", false), equals, "replacement")
}

func (s *Suite) TestMkopSubst_all(c *check.C) {
	c.Assert(mkopSubst("aaaaa", false, "a", false, "b", true), equals, "bbbbb")
	c.Assert(mkopSubst("aaaaa", true, "a", false, "b", true), equals, "baaaa")
	c.Assert(mkopSubst("aaaaa", false, "a", true, "b", true), equals, "aaaab")
	c.Assert(mkopSubst("aaaaa", true, "a", true, "b", true), equals, "aaaaa")
}
