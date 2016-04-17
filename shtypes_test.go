package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) Test_ShToken_String(c *check.C) {
	c.Check(shtBacktClose.String(), equals, "backtClose")
	c.Check(shtComment.String(), equals, "comment")
}

func (s *Suite) Test_ShQuoting_String(c *check.C) {
	c.Check(shqUnknown.String(), equals, "unknown")
}
