package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVartypeEffectivePermissions(c *check.C) {
	G.globalData.InitVartypes()

	{
		t := G.globalData.vartypes["PREFIX"]

		c.Check(t.checker.name, equals, "Pathname")
		c.Check(t.aclEntries, check.DeepEquals, []AclEntry{{glob: "*", permissions: aclpUse}})
		c.Check(t.effectivePermissions("Makefile"), equals, aclpUse)
	}

	{
		t := G.globalData.vartypes["EXTRACT_OPTS"]

		c.Check(t.checker.name, equals, "ShellWord")
		c.Check(t.effectivePermissions("Makefile"), equals, aclpAppend|aclpSet)
		c.Check(t.effectivePermissions("../Makefile"), equals, aclpAppend|aclpSet)
		c.Check(t.effectivePermissions("options.mk"), equals, aclpUnknown)
	}
}

func (s *Suite) TestVarCheckerHasEnum(c *check.C) {
	vc := enum("catinstall middle maninstall")

	c.Check(vc.HasEnum("catinstall"), equals, true)
	c.Check(vc.HasEnum("middle"), equals, true)
	c.Check(vc.HasEnum("maninstall"), equals, true)
}

func (s *Suite) TestAclPermissions_contains(c *check.C) {
	perms := aclpAllRuntime

	c.Check(perms.contains(aclpAllRuntime), equals, true)
	c.Check(perms.contains(aclpUse), equals, true)
	c.Check(perms.contains(aclpUseLoadtime), equals, false)
}

func (s *Suite) TestAclPermissions_String(c *check.C) {
	c.Check(AclPermissions(0).String(), equals, "none")
	c.Check(aclpAll.String(), equals, "append, default, preprocess, set, runtime")
	c.Check(aclpUnknown.String(), equals, "unknown")
}
