package licenses

import (
	"gopkg.in/check.v1"
	"testing"
)

type Suite struct{}

func (s *Suite) Test_Parse(c *check.C) {
	c.Check(Parse("gnu-gpl-v2"), check.DeepEquals, &Condition{Name: "gnu-gpl-v2"})

	c.Check(Parse("a AND b AND c AND d").String(), check.Equals, "(a AND b AND c AND d)")
	c.Check(
		Parse("a AND b AND c AND d"),
		check.DeepEquals,
		&Condition{Name: "a",
			And: []*Condition{{Name: "b"}, {Name: "c"}, {Name: "d"}}})

	c.Check(Parse("a OR b OR c OR d").String(), check.Equals, "(a OR b OR c OR d)")
	c.Check(
		Parse("a OR b OR c OR d"),
		check.DeepEquals,
		&Condition{Name: "a",
			Or: []*Condition{{Name: "b"}, {Name: "c"}, {Name: "d"}}})

	c.Check(Parse("(a OR b) AND (c AND d)").String(), check.Equals, "((a OR b) AND (c AND d))")
	c.Check(
		(Parse("(a OR b) AND (c AND d)")),
		check.DeepEquals,
		&Condition{
			Main: &Condition{
				Name: "a",
				Or:   []*Condition{{Name: "b"}}},
			And: []*Condition{{Main: &Condition{
				Name: "c",
				And:  []*Condition{{Name: "d"}}}}}})

	c.Check(Parse("AND artistic"), check.IsNil)
}
func Test(t *testing.T) {
	check.Suite(new(Suite))
	check.TestingT(t)
}
