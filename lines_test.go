package main

import "gopkg.in/check.v1"

func (s *Suite) Test_Lines_CheckRcsid(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("filename",
		"$"+"NetBSD: dummy $",
		"$"+"NetBSD$",
		"$"+"Id: dummy $",
		"$"+"Id$",
		"$"+"FreeBSD$")

	for i := range lines.Lines {
		lines.CheckRcsid(i, ``, "")
	}

	t.CheckOutputLines(
		"ERROR: filename:3: Expected \"$"+"NetBSD$\".",
		"ERROR: filename:4: Expected \"$"+"NetBSD$\".",
		"ERROR: filename:5: Expected \"$"+"NetBSD$\".")
}
