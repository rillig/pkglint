package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_Var_Literal(c *check.C) {
	v := NewVar("VARNAME")

	// FIXME: Replace this test with an actual use case.

	c.Check(v.Literal(), equals, false)
}

func (s *Suite) ignored_Test_Var_LiteralValue(c *check.C) {
	v := NewVar("VARNAME")

	// FIXME: Replace this test with an actual use case.

	c.Check(v.LiteralValue(), equals, "")
}
