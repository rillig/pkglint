package main

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_RawLine_String(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("filename", 123, "text")

	c.Check(line.raw[0].String(), equals, "123:text\n")
}
