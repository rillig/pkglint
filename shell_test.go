package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestSplitIntoShellwords_LineContinuation(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)

	words, rest := splitIntoShellwords(line, "if true; then \\")

	c.Check(words, check.DeepEquals, []string{"if", "true", ";", "then"})
	c.Check(rest, equals, "\\")

	words, rest = splitIntoShellwords(line, "pax -s /.*~$$//g")

	c.Check(words, check.DeepEquals, []string{"pax", "-s", "/.*~$$//g"})
	c.Check(rest, equals, "")
}

func (s *Suite) TestChecklineMkShelltext(c *check.C) {
	G.mkContext = newMkContext()
	line := NewLine("fname", "1", "dummy", nil)

	NewMkShellLine(line).checklineMkShelltext("@# Comment")
}

func (s *Suite) TestChecklineMkShellword(c *check.C) {
	UseCommandLine("-Wall")
	line := NewLine("fname", "1", "dummy", nil)
	
	c.Check(matches("${list}", `^`+reVarname+`$`), equals, true)
	c.Check(matches("${list}", `^`+reVarnameDirect+`$`), equals, false)
	
	checklineMkShellword(line, "${${list}}", false)
	
	c.Check(s.Output(), equals, "")

	checklineMkShellword(line, "\"$@\"", false)
	
	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"${.TARGET}\" instead of \"$@\".\n")
}
