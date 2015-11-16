package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestGlobalDataVartypes(c *check.C) {
	G.globalData.InitVartypes()

	c.Check(G.globalData.vartypes["BSD_MAKE_ENV"].checker.name, equals, "ShellWord")
	c.Check(G.globalData.vartypes["USE_BUILTIN.*"].checker.name, equals, "YesNo_Indirectly")
}
