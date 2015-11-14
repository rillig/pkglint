package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVartypeEffectivePermissions(c *check.C) {
	initacls()

	{
		t := aclVartypes["PREFIX"]

		c.Check(t.checker.name, equals, "Pathname")
		c.Check(t.aclEntries, check.DeepEquals, []AclEntry{{glob: "*", permissions: "u"}})
		c.Check(t.effectivePermissions("Makefile"), equals, "u")
	}

	{
		t := aclVartypes["EXTRACT_OPTS"]

		c.Check(t.checker.name, equals, "ShellWord")
		c.Check(t.effectivePermissions("Makefile"), equals, "as")
		c.Check(t.effectivePermissions("../Makefile"), equals, "as")
		c.Check(t.effectivePermissions("options.mk"), equals, "")
	}
}
