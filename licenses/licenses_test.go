package licenses

import (
	"gopkg.in/check.v1"
	"testing"
)

type Suite struct{}

func (s *Suite) Test_Parse(c *check.C) {
	c.Check(Parse("gnu-gpl-v2"), check.DeepEquals, &Condition{Name: "gnu-gpl-v2"})

	c.Check(Parse("a AND b").String(), check.Equals, "(a AND b)")
	c.Check(Parse("a OR b").String(), check.Equals, "(a OR b)")

	c.Check(Parse("a OR (b AND c)").String(), check.Equals, "(a OR (b AND c))")
	c.Check(Parse("(a OR b) AND c").String(), check.Equals, "((a OR b) AND c)")

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

func (s *Suite) Test_Condition_String(c *check.C) {
	name := func(name string) *Condition {
		return &Condition{Name: name}
	}
	and := func(parts ...*Condition) *Condition {
		return &Condition{Main: parts[0], And: parts[1:]}
	}
	or := func(parts ...*Condition) *Condition {
		return &Condition{Main: parts[0], Or: parts[1:]}
	}

	c.Check(name("a").String(), check.Equals, "a")
	c.Check(and(name("a"), name("b")).String(), check.Equals, "(a AND b)")
	c.Check(or(name("a"), name("b")).String(), check.Equals, "(a OR b)")
	c.Check(and(or(name("a"), name("b")), or(name("c"), name("d"))).String(), check.Equals, "((a OR b) AND (c OR d))")
}

func Test(t *testing.T) {
	check.Suite(new(Suite))
	check.TestingT(t)
}
