package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_Var_LiteralValue__assign(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.LiteralValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\toverwritten"))

	t.Check(v.LiteralValue(), equals, "overwritten")
}

func (s *Suite) Test_Var_LiteralValue__assign_reference(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.LiteralValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\t${OTHER}"))

	t.Check(v.Literal(), equals, false)
}

func (s *Suite) Test_Var_LiteralValue__assign_conditional(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	t.Check(v.ConditionalVars(), check.IsNil)

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.Literal(), equals, false)
}

func (s *Suite) Test_Var_LiteralValue__default(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME?=\tvalue"))

	t.Check(v.LiteralValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME?=\tignored"))

	t.Check(v.LiteralValue(), equals, "value")
}

func (s *Suite) Test_Var_LiteralValue__append(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME+=\tvalue"))

	t.Check(v.LiteralValue(), equals, " value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME+=\tappended"))

	t.Check(v.LiteralValue(), equals, " value appended")
}

func (s *Suite) Test_Var_LiteralValue__eval(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME:=\tvalue"))

	t.Check(v.LiteralValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME:=\toverwritten"))

	t.Check(v.LiteralValue(), equals, "overwritten")
}

// Variables that are based on running shell commands are never literal.
func (s *Suite) Test_Var_LiteralValue__shell(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tvalue"))

	t.Check(v.LiteralValue(), equals, "value")

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME!=\techo hello"))

	t.Check(v.Literal(), equals, false)
}

func (s *Suite) Test_Var_ConditionalVars(c *check.C) {
	t := s.Init(c)

	v := NewVar("VARNAME")

	t.Check(v.ConditionalVars(), check.IsNil)

	v.Write(t.NewMkLine("write.mk", 123, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.Literal(), equals, false)
	t.Check(v.ConditionalVars(), deepEquals, []string{"OPSYS"})

	v.Write(t.NewMkLine("write.mk", 124, "VARNAME=\tconditional"), "OPSYS")

	t.Check(v.ConditionalVars(), deepEquals, []string{"OPSYS"})
}
