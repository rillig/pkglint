package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) Test_ShLexeme_String(c *check.C) {
	c.Check(shlBacktClose.String(), equals, "backtClose")
	c.Check(shlComment.String(), equals, "comment")
}
