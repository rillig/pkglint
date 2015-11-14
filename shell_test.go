package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestSplitIntoShellwords_LineContinuation(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)

	words, rest := splitIntoShellwords(line, "if true; then \\")

	c.Check(words, check.DeepEquals, []string{"if", "true", ";", "then"})
	c.Check(rest, equals, "\\")
}
