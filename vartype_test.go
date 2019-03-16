package pkglint

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_Vartype_EffectivePermissions(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	if typ := G.Pkgsrc.vartypes.Canon("PREFIX"); c.Check(typ, check.NotNil) {
		c.Check(typ.basicType.name, equals, "Pathname")
		c.Check(typ.aclEntries, check.DeepEquals, []ACLEntry{{"*", aclpUse}})
		c.Check(typ.EffectivePermissions("Makefile"), equals, aclpUse)
		c.Check(typ.EffectivePermissions("buildlink3.mk"), equals, aclpUse)
	}

	if typ := G.Pkgsrc.vartypes.Canon("EXTRACT_OPTS"); c.Check(typ, check.NotNil) {
		c.Check(typ.basicType.name, equals, "ShellWord")
		c.Check(typ.EffectivePermissions("Makefile"), equals, aclpAllWrite|aclpUse)
		c.Check(typ.EffectivePermissions("options.mk"), equals, aclpAllWrite|aclpUse)
	}
}

func (s *Suite) Test_Vartype_AlternativeFiles(c *check.C) {
	t := s.Init(c)

	G.Pkgsrc.vartypes.DefineParse("VAR", lkNone, BtYesNo,
		"buildlink3.mk: none",
		"special:b*.mk: use",
		"*.mk: none",
		"Makefile: use",
		"Makefile.*: none",
		"*: use")
	G.Pkgsrc.vartypes.DefineParse("ONLY", lkNone, BtYesNo,
		"buildlink3.mk: append",
		"*: none")

	test := func(varname string, perms ACLPermissions, alternatives string) {
		vartype := G.Pkgsrc.VariableType(varname)
		t.Check(vartype.AlternativeFiles(perms), equals, alternatives)
	}

	// Just out of curiosity, since calling the function with no
	// permissions doesn't make sense. The result is the complete
	// file list.
	test("VAR", aclpNone, "buildlink3.mk, b*.mk, *.mk, Makefile, Makefile.* or *")

	test("VAR", aclpUse, "b*.mk, Makefile or *, but not buildlink3.mk, *.mk or Makefile.*")

	test("VAR", aclpUseLoadtime, "")

	test("ONLY", aclpAppend, "buildlink3.mk only")
}

func (s *Suite) Test_BasicType_HasEnum(c *check.C) {
	vc := enum("start middle end")

	c.Check(vc.HasEnum("start"), equals, true)
	c.Check(vc.HasEnum("middle"), equals, true)
	c.Check(vc.HasEnum("end"), equals, true)

	c.Check(vc.HasEnum("star"), equals, false)
	c.Check(vc.HasEnum("mid"), equals, false)
	c.Check(vc.HasEnum("nd"), equals, false)
	c.Check(vc.HasEnum("start middle"), equals, false)
}

func (s *Suite) Test_ACLPermissions_Contains(c *check.C) {
	perms := aclpAllRuntime

	c.Check(perms.Contains(aclpAllRuntime), equals, true)
	c.Check(perms.Contains(aclpUse), equals, true)
	c.Check(perms.Contains(aclpUseLoadtime), equals, false)
}

func (s *Suite) Test_ACLPermissions_String(c *check.C) {
	c.Check(ACLPermissions(0).String(), equals, "none")
	c.Check(aclpAll.String(), equals, "set, set-default, append, use-loadtime, use")
}

func (s *Suite) Test_ACLPermissions_HumanString(c *check.C) {

	c.Check(ACLPermissions(0).HumanString(),
		equals, "") // Doesn't happen in practice

	c.Check(aclpAll.HumanString(),
		equals, "set, given a default value, appended to, used at load time, or used")
}

func (s *Suite) Test_Vartype_IsConsideredList(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	c.Check(G.Pkgsrc.VariableType("COMMENT").IsConsideredList(), equals, false)
	c.Check(G.Pkgsrc.VariableType("DEPENDS").IsConsideredList(), equals, true)
	c.Check(G.Pkgsrc.VariableType("PKG_FAIL_REASON").IsConsideredList(), equals, true)
	c.Check(G.Pkgsrc.VariableType("CONF_FILES").IsConsideredList(), equals, true)
}
