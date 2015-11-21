package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestGetopt_Short(c *check.C) {
	opts := NewOptions()
	var help bool
	opts.AddFlagVar('h', "help", &help, false, "prints a help page")

	args, err := opts.Parse([]string{"progname", "-h"})

	c.Assert(err, check.IsNil)
	c.Check(args, check.DeepEquals, []string{})
	c.Check(help, equals, true)
}

func (s *Suite) TestGetopt_UnknownShort(c *check.C) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "-z"})

	c.Check(err.Error(), equals, "progname: unknown option: -z")
}

func (s *Suite) TestGetopt_UnknownLong(c *check.C) {
	opts := NewOptions()

	_, err := opts.Parse([]string{"progname", "--unknown-long"})

	c.Check(err.Error(), equals, "progname: unknown option: --unknown-long")
}

func (s *Suite) TestGetopt_UnknownFlag(c *check.C) {
	opts := NewOptions()
	opts.AddFlagGroup('W', "warnings", "", "")

	_, err := opts.Parse([]string{"progname", "-Wall", "-Werror"})

	c.Check(err.Error(), equals, "progname: unknown option: -Werror")

	_, err = opts.Parse([]string{"progname", "--warnings=all", "--warnings=no-error"})

	c.Check(err.Error(), equals, "progname: unknown option: --warnings=no-error")
}
