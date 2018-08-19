package main

import "gopkg.in/check.v1"

func (s *Suite) Test_Tools_ParseToolLine(c *check.C) {
	t := s.Init(c)

	t.SetupToolUsable("tool1", "")
	t.SetupVartypes()
	t.SetupFileLines("Makefile",
		MkRcsID,
		"",
		"USE_TOOLS.NetBSD+=\ttool1")

	CheckdirToplevel(t.File("."))

	// No error about "Unknown tool \"NetBSD\"."
	t.CheckOutputEmpty()
}

func (s *Suite) Test_Tools_validateToolName__invalid(c *check.C) {
	t := s.Init(c)

	reg := NewTools()

	reg.DefineName("tool_name", dummyMkLine)
	reg.DefineName("tool:dependency", dummyMkLine)
	reg.DefineName("tool:build", dummyMkLine)

	// Currently, the underscore is not used in any tool name.
	// If there should ever be such a case, just use a different character for testing.
	t.CheckOutputLines(
		"ERROR: Invalid tool name \"tool_name\".",
		"ERROR: Invalid tool name \"tool:dependency\".",
		"ERROR: Invalid tool name \"tool:build\".")
}
