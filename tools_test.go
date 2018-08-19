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

	reg.DefineName("tool_name", dummyMkLine, false)
	reg.DefineName("tool:dependency", dummyMkLine, false)
	reg.DefineName("tool:build", dummyMkLine, false)

	// Currently, the underscore is not used in any tool name.
	// If there should ever be such a case, just use a different character for testing.
	t.CheckOutputLines(
		"ERROR: Invalid tool name \"tool_name\".",
		"ERROR: Invalid tool name \"tool:dependency\".",
		"ERROR: Invalid tool name \"tool:build\".")
}

func (s *Suite) Test_Tools_Trace__coverage(c *check.C) {
	t := s.Init(c)

	t.DisableTracing()

	reg := NewTools()
	reg.Trace()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_Tools__USE_TOOLS_predefined_sed(c *check.C) {
	t := s.Init(c)

	t.SetupPkgsrc()
	t.CreateFileLines("mk/bsd.prefs.mk",
		MkRcsID,
		"",
		"USE_TOOLS+=\tsed:pkgsrc")
	t.SetupFileMkLines("module.mk",
		MkRcsID,
		"",
		"do-build:",
		"\t${SED} < input > output",
		"\t${AWK} < input > output")

	G.Main("pkglint", "-Wall", t.File("module.mk"))

	t.CheckOutputLines(
		"WARN: ~/module.mk:4: Unknown shell command \"${SED}\".", // FIXME: sed is added to USE_TOOLS.
		"WARN: ~/module.mk:5: Unknown shell command \"${AWK}\".",
		"0 errors and 2 warnings found.",
		"(Run \"pkglint -e\" to show explanations.)")
}
