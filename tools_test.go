package main

import "gopkg.in/check.v1"

func (s *Suite) Test_USE_TOOLS(c *check.C) {
	t := s.Init(c)

	t.SetupTool(&Tool{Name: "tool1", Predefined: true})
	t.SetupTool(&Tool{Name: "tool2", Predefined: true})
	t.SetupTool(&Tool{Name: "tool3", Predefined: true})
	t.SetupVartypes()
	t.SetupFileLines("Makefile",
		MkRcsID,
		"",
		"USE_TOOLS+=\ttool1",
		"USE_TOOLS+=\ttool2:pkgsrc",
		"USE_TOOLS+=\ttool3:run",
		"USE_TOOLS+=\ttool4:unknown")

	G.CurrentDir = t.TmpDir()
	CheckdirToplevel()

	t.CheckOutputLines(
		"ERROR: ~/Makefile:6: Unknown tool \"tool4\".",
		"ERROR: ~/Makefile:6: Unknown tool dependency \"unknown\". Use one of \"bootstrap\", \"build\", \"pkgsrc\" or \"run\".")
}
