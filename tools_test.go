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

	reg.Define("tool_name", "", dummyMkLine, false)
	reg.Define("tool:dependency", "", dummyMkLine, false)
	reg.Define("tool:build", "", dummyMkLine, false)

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
	t.CreateFileLines("mk/tools/defaults.mk",
		MkRcsID,
		"",
		"_TOOLS_VARNAME.sed=\tSED")
	t.SetupFileMkLines("module.mk",
		MkRcsID,
		"",
		"do-build:",
		"\t${SED} < input > output",
		"\t${AWK} < input > output")

	G.Main("pkglint", "-Wall", t.File("module.mk"))

	t.CheckOutputLines(
		"WARN: ~/module.mk:5: Unknown shell command \"${AWK}\".",
		"0 errors and 1 warning found.",
		"(Run \"pkglint -e\" to show explanations.)")
}

func (s *Suite) Test_Tools__add_varname_later(c *check.C) {

	tools := NewTools()
	tool := tools.Define("tool", "", dummyMkLine, true)

	c.Check(tool.Name, equals, "tool")
	c.Check(tool.Varname, equals, "")

	// Should update the existing tool definition.
	tools.Define("tool", "TOOL", dummyMkLine, true)

	c.Check(tool.Name, equals, "tool")
	c.Check(tool.Varname, equals, "TOOL")
}

func (s *Suite) Test_Tools__load_from_infrastructure(c *check.C) {
	t := s.Init(c)

	tools := NewTools()

	t.NewMkLines("create.mk",
		"TOOLS_CREATE+= load",
		"TOOLS_CREATE+= run",
		"TOOLS_CREATE+= nowhere",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	// The references to the tools are stable,
	// the lookup methods always return the same objects.
	load := tools.ByNameTool("load")
	run := tools.ByNameTool("run")
	nowhere := tools.ByNameTool("nowhere")

	// All tools are defined by name, but their variable names are not yet known.
	// At this point they may not be used, neither by the pkgsrc infrastructure nor by a package.
	c.Check(load, deepEquals, &Tool{"load", "", false, Nowhere})
	c.Check(run, deepEquals, &Tool{"run", "", false, Nowhere})
	c.Check(nowhere, deepEquals, &Tool{"nowhere", "", false, Nowhere})

	t.NewMkLines("varnames.mk",
		"_TOOLS_VARNAME.load=    LOAD",
		"_TOOLS_VARNAME.run=     RUN_CMD", // To avoid a collision with ${RUN}.
		"_TOOLS_VARNAME.nowhere= NOWHERE",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	// At this point the tools can be found by their variable names, too.
	// They still may not be used.
	c.Check(load, deepEquals, &Tool{"load", "LOAD", false, Nowhere})
	c.Check(run, deepEquals, &Tool{"run", "RUN_CMD", false, Nowhere})
	c.Check(nowhere, deepEquals, &Tool{"nowhere", "NOWHERE", false, Nowhere})
	c.Check(tools.ByVarnameTool("LOAD"), equals, load)
	c.Check(tools.ByVarnameTool("RUN_CMD"), equals, run)
	c.Check(tools.ByVarnameTool("NOWHERE"), equals, nowhere)
	c.Check(load.UsableAtLoadTime(false), equals, false)
	c.Check(load.UsableAtLoadTime(true), equals, false)
	c.Check(load.UsableAtRunTime(), equals, false)
	c.Check(run.UsableAtLoadTime(false), equals, false)
	c.Check(run.UsableAtLoadTime(true), equals, false)
	c.Check(run.UsableAtRunTime(), equals, false)
	c.Check(nowhere.UsableAtLoadTime(false), equals, false)
	c.Check(nowhere.UsableAtLoadTime(true), equals, false)
	c.Check(nowhere.UsableAtRunTime(), equals, false)

	t.NewMkLines("bsd.prefs.mk",
		"USE_TOOLS+= load",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	// Tools that are added to USE_TOOLS in bsd.prefs.mk may be used afterwards.
	// By variable name, they may be used both at load time as well as run time.
	// By plain name, they may be used only in {pre,do,post}-* targets.
	c.Check(load, deepEquals, &Tool{"load", "LOAD", false, AfterPrefsMk})
	c.Check(load.UsableAtLoadTime(false), equals, false)
	c.Check(load.UsableAtLoadTime(true), equals, true)
	c.Check(load.UsableAtRunTime(), equals, true)

	t.NewMkLines("bsd.pkg.mk",
		"USE_TOOLS+= run",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	// Tools that are added to USE_TOOLS in bsd.pkg.mk may be used afterwards at run time.
	// The {pre,do,post}-* targets may use both forms (${CAT} and cat).
	// All other targets must use the variable form (${CAT}).
	c.Check(run, deepEquals, &Tool{"run", "RUN_CMD", false, AtRunTime})
	c.Check(run.UsableAtLoadTime(false), equals, false)
	c.Check(run.UsableAtLoadTime(false), equals, false)
	c.Check(run.UsableAtRunTime(), equals, true)

	// That's all for parsing tool definitions from the pkgsrc infrastructure.
}

func (s *Suite) Test_Tools__package_Makefile(c *check.C) {
	t := s.Init(c)

	// The same tools as in Test_Tools__load_from_infrastructure.
	t.SetupPkgsrc()
	t.CreateFileLines("mk/tools/defaults.mk",
		"TOOLS_CREATE+=  load",
		"TOOLS_CREATE+=  run",
		"TOOLS_CREATE+=  nowhere",
		"TOOLS_CREATE+=  pkg-before-prefs",
		"TOOLS_CREATE+=  pkg-after-prefs",
		"_TOOLS_VARNAME.load=                    LOAD",
		"_TOOLS_VARNAME.run=                     RUN_CMD",
		"_TOOLS_VARNAME.nowhere=                 NOWHERE",
		"_TOOLS_VARNAME.pkg-before-prefs=        PKG_BEFORE_PREFS",
		"_TOOLS_VARNAME.pkg-after-prefs=         PKG_AFTER_PREFS",
	)
	t.CreateFileLines("mk/bsd.prefs.mk",
		"USE_TOOLS+=     load")
	t.CreateFileLines("mk/bsd.pkg.mk",
		"USE_TOOLS+=     run")
	G.Pkgsrc.LoadInfrastructure()

	tools := NewTools()
	tools.AddAll(G.Pkgsrc.Tools)

	load := tools.ByNameTool("load")
	run := tools.ByNameTool("run")
	nowhere := tools.ByNameTool("nowhere")
	before := tools.ByNameTool("pkg-before-prefs")
	after := tools.ByNameTool("pkg-after-prefs")

	c.Check(load.UsableAtRunTime(), equals, false) // FIXME: must be true, since it is added in bsd.prefs.mk.
	c.Check(run.UsableAtRunTime(), equals, false)  // FIXME: must be true, since it is added in bsd.pkg.mk.
	c.Check(nowhere.UsableAtRunTime(), equals, false)

	// The seenPrefs variable is only relevant for the package Makefile.
	// All other files must not use the tools at load time.
	// For them, seenPrefs can be though of as being true from the beginning.

	t.NewMkLines("Makefile",
		"USE_TOOLS+=     pkg-before-prefs",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	c.Check(before.Validity, equals, AtRunTime) // FIXME: must be AfterPrefsMk
	c.Check(before.UsableAtLoadTime(false), equals, false)
	c.Check(before.UsableAtLoadTime(true), equals, false) // FIXME: must be true
	c.Check(before.UsableAtRunTime(), equals, true)

	c.Check(tools.SeenPrefs, equals, false)

	t.NewMkLines("Makefile",
		".include \"../../mk/bsd.prefs.mk\"",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	c.Check(tools.SeenPrefs, equals, true)

	t.NewMkLines("Makefile",
		"USE_TOOLS+=     pkg-after-prefs",
	).ForEach(func(mkline MkLine) {
		tools.ParseToolLine(mkline, false)
	})

	c.Check(after.Validity, equals, AtRunTime)
	c.Check(after.UsableAtLoadTime(false), equals, false)
	c.Check(after.UsableAtLoadTime(true), equals, false)
	c.Check(after.UsableAtRunTime(), equals, true)
}
