package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestGetopt_UnknownShort(c *check.C) {
	opts := NewOptions(G.logOut)

	_, err := opts.Parse([]string{"progname", "-z"})

	c.Check(err.Error(), equals, "unknown option: -z")
}

func (s *Suite) TestGetopt_UnknownLong(c *check.C) {
	opts := NewOptions(G.logOut)

	_, err := opts.Parse([]string{"progname", "--unknown-long"})

	c.Check(err.Error(), equals, "unknown option: --unknown-long")
}

func (s *Suite) TestGetopt_UnknownFlag(c *check.C) {
	opts := NewOptions(G.logOut)
	opts.AddFlagGroup('W', "warnings", "", "")

	_, err := opts.Parse([]string{"progname", "-Wall", "-Werror"})

	c.Check(err.Error(), equals, "unknown option: -Werror")

	_, err = opts.Parse([]string{"progname", "--warnings=all", "--warnings=no-error"})

	c.Check(err.Error(), equals, "unknown option: --warnings=no-error")
}
