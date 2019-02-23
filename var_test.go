package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_Var_ConstantValue__assign(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\toverwritten"))

	t.Check(v.ConstantValue(), equals, "overwritten")
}

func (s *Suite) Test_Var_ConstantValue__assign_reference(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\t${OTHER}"))

	t.Check(v.Constant(), equals, false)
}

func (s *Suite) Test_Var_ConstantValue__assign_conditional(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	t.Check(v.ConditionalVars(), check.IsNil)

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.Constant(), equals, false)
}

func (s *Suite) Test_Var_ConstantValue__default(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME?=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME?=\tignored"))

	t.Check(v.ConstantValue(), equals, "value")
}

func (s *Suite) Test_Var_ConstantValue__append(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME+=\tvalue"))

	t.Check(v.ConstantValue(), equals, " value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME+=\tappended"))

	t.Check(v.ConstantValue(), equals, " value appended")
}

func (s *Suite) Test_Var_ConstantValue__eval(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME:=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten"))

	t.Check(v.ConstantValue(), equals, "overwritten")
}

// Variables that are based on running shell commands are never constant.
func (s *Suite) Test_Var_ConstantValue__shell(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME!=\techo hello"))

	t.Check(v.Constant(), equals, false)
}

func (s *Suite) Test_Var_ConstantValue__referenced_before(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	// Since the value of VARNAME escapes here, the value is not
	// guaranteed to be the same in all evaluations of ${VARNAME}.
	// For example, OTHER may be used at load time in an .if
	// condition.
	v.Read(t.NewMkLine("readwrite.mk", 123, "OTHER=\t${VARNAME}"))

	t.Check(v.Constant(), equals, false)

	v.Write(t.NewMkLine("readwrite.mk", 124, "VARNAME=\tvalue"))

	t.Check(v.Constant(), equals, false)
}

func (s *Suite) Test_Var_ConstantValue__referenced_in_between(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("readwrite.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.ConstantValue(), equals, "value")

	// Since the value of VARNAME escapes here, the value is not
	// guaranteed to be the same in all evaluations of ${VARNAME}.
	// For example, OTHER may be used at load time in an .if
	// condition.
	v.Read(t.NewMkLine("readwrite.mk", 124, "OTHER=\t${VARNAME}"))

	t.Check(v.ConstantValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 125, "VARNAME=\toverwritten"))

	t.Check(v.Constant(), equals, false)
}

func (s *Suite) Test_Var_ConditionalVars(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	t.Check(v.Conditional(), equals, false)
	t.Check(v.ConditionalVars(), check.IsNil)

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.Constant(), equals, false)
	t.Check(v.Conditional(), equals, true)
	t.Check(v.ConditionalVars(), deepEquals, []string{"OPSYS"})

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.Conditional(), equals, true)
	t.Check(v.ConditionalVars(), deepEquals, []string{"OPSYS"})
}

func (s *Suite) Test_Var_Value__initial_conditional_write(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten conditionally"), "OPSYS")

	// Since there is no previous value, the simplest choice is to just
	// take the first seen value, no matter if that value is conditional
	// or not.
	t.Check(v.Conditional(), equals, true)
	t.Check(v.Constant(), equals, false)
	t.Check(v.Value(), equals, "overwritten conditionally")
}

func (s *Suite) Test_Var_Value__conditional_write_after_unconditional(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.Value(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME+=\tappended"))

	t.Check(v.Value(), equals, "value appended")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten conditionally"), "OPSYS")

	// When there is a previous value, it's probably best to keep
	// that value since this way the following code results in the
	// most generic value:
	//  VAR=    generic
	//  .if ${OPSYS} == NetBSD
	//  VAR=    specific
	//  .endif
	// The value stays the same, still it is marked as conditional and therefore
	// not constant anymore.
	t.Check(v.Conditional(), equals, true)
	t.Check(v.Constant(), equals, false)
	t.Check(v.Value(), equals, "value appended")
}

func (s *Suite) Test_Var_Value__infrastructure(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine(t.File("write.mk"), 123, "VARNAME=\tvalue"))

	t.Check(v.Value(), equals, "value")

	v.Write(t.NewMkLine(t.File("mk/write.mk"), 123, "VARNAME=\tinfra"))

	t.Check(v.Value(), equals, "value")

	v.Write(t.NewMkLine(t.File("wip/mk/write.mk"), 123, "VARNAME=\twip infra"))

	t.Check(v.Value(), equals, "value")
}

func (s *Suite) Test_Var_ValueInfra(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine(t.File("write.mk"), 123, "VARNAME=\tvalue"))

	t.Check(v.ValueInfra(), equals, "value")

	v.Write(t.NewMkLine(t.File("mk/write.mk"), 123, "VARNAME=\tinfra"))

	t.Check(v.ValueInfra(), equals, "infra")

	v.Write(t.NewMkLine(t.File("wip/mk/write.mk"), 123, "VARNAME=\twip infra"))

	t.Check(v.ValueInfra(), equals, "wip infra")
}

func (s *Suite) Test_Var_ReadLocations(c *check.C) {
	t := s.Init(c)

	v := NewVar("VAR")

	t.Check(v.ReadLocations(), check.IsNil)

	mkline123 := t.NewMkLine("read.mk", 123, "OTHER=\t${VAR}")
	v.Read(mkline123)

	t.Check(v.ReadLocations(), deepEquals, []MkLine{mkline123})

	mkline124 := t.NewMkLine("read.mk", 124, "OTHER=\t${VAR} ${VAR}")
	v.Read(mkline124)
	v.Read(mkline124)

	// For now, count every read of the variable. I'm not yet sure
	// whether that's the best way or whether to make the lines unique.
	t.Check(v.ReadLocations(), deepEquals, []MkLine{mkline123, mkline124, mkline124})
}

func (s *Suite) Test_Var_WriteLocations(c *check.C) {
	t := s.Init(c)

	v := NewVar("VAR")

	t.Check(v.WriteLocations(), check.IsNil)

	mkline123 := t.NewMkLine("write.mk", 123, "VAR=\tvalue")
	v.Write(mkline123)

	t.Check(v.WriteLocations(), deepEquals, []MkLine{mkline123})

	// Multiple writes from the same line may happen because of a .for loop.
	mkline125 := t.NewMkLine("write.mk", 125, "VAR+=\t${var}")
	v.Write(mkline125)
	v.Write(mkline125)

	// For now, count every write of the variable. I'm not yet sure
	// whether that's the best way or whether to make the lines unique.
	t.Check(v.WriteLocations(), deepEquals, []MkLine{mkline123, mkline125, mkline125})
}

func (s *Suite) Test_Var_Refs(c *check.C) {
	t := s.Init(c)

	v := NewVar("VAR")

	t.Check(v.Refs(), check.IsNil)

	v.Write(t.NewMkLine("write.mk", 123, "VAR=${OTHER} ${${OPSYS} == NetBSD :? ${THEN} : ${ELSE}}"), "COND")

	v.AddRef("FOR")

	t.Check(v.Refs(), deepEquals, []string{"OTHER", "OPSYS", "THEN", "ELSE", "COND", "FOR"})
}
