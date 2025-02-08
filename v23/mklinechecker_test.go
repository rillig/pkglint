package pkglint

import (
	"gopkg.in/check.v1"
	"runtime"
)

// PR pkg/46570, item 2
func (s *Suite) Test_MkLineChecker__unclosed_expr(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"EGDIRS=\t${EGDIR/apparmor.d ${EGDIR/dbus-1/system.d ${EGDIR/pam.d")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:2: Missing closing \"}\" for \"EGDIR/pam.d\".",
		"WARN: Makefile:2: Invalid part \"/pam.d\" after variable name \"EGDIR\".",
		"WARN: Makefile:2: Missing closing \"}\" for \"EGDIR/dbus-1/system.d ${EGDIR/pam.d\".",
		"WARN: Makefile:2: Invalid part \"/dbus-1/system.d ${EGDIR/pam.d\" after variable name \"EGDIR\".",
		"WARN: Makefile:2: Missing closing \"}\" for \"EGDIR/apparmor.d ${EGDIR/dbus-1/system.d ${EGDIR/pam.d\".",
		"WARN: Makefile:2: Invalid part \"/apparmor.d ${EGDIR/dbus-1/system.d ${EGDIR/pam.d\" after variable name \"EGDIR\".",
		"WARN: Makefile:2: Variable \"EGDIRS\" is defined but not used.",
		"WARN: Makefile:2: Variable \"EGDIR/pam.d\" is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_Check__buildlink3_include_prefs(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	t.CreateFileLines("mk/bsd.prefs.mk")
	t.CreateFileLines("mk/bsd.fast.prefs.mk")
	mklines := t.SetUpFileMkLines("category/package/buildlink3.mk",
		MkCvsID,
		".include \"../../mk/bsd.prefs.mk\"",
		".include \"../../mk/bsd.fast.prefs.mk\"")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/category/package/buildlink3.mk:2: For efficiency reasons, " +
			"include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
}

func (s *Suite) Test_MkLineChecker_Check__warn_expr_LOCALBASE(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("options.mk",
		MkCvsID,
		"PKGNAME=\t${LOCALBASE}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: options.mk:2: Use PREFIX instead of LOCALBASE.")
}

func (s *Suite) Test_MkLineChecker_Check__expr_modifier_L(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("x11/xkeyboard-config/Makefile",
		MkCvsID,
		"FILES_SUBST+=\tXKBCOMP_SYMLINK=${${XKBBASE}/xkbcomp:L:Q}",
		"FILES_SUBST+=\tXKBCOMP_SYMLINK=${${XKBBASE}/xkbcomp:Q}")

	mklines.Check()

	// In line 2, don't warn that ${XKBBASE}/xkbcomp is used but not defined.
	// This is because the :L modifier interprets everything before as an expression
	// instead of a variable name.
	//
	// In line 3, the :L modifier is missing, therefore ${XKBBASE}/xkbcomp is the
	// name of another variable, and that variable is not known. Only XKBBASE is known.
	//
	// In line 3, warn about the invalid "/" as part of the variable name.
	t.CheckOutputLines(
		"WARN: x11/xkeyboard-config/Makefile:3: "+
			"Invalid part \"/xkbcomp\" after variable name \"${XKBBASE}\".",
		// TODO: Avoid these duplicate diagnostics.
		"WARN: x11/xkeyboard-config/Makefile:3: "+
			"Invalid part \"/xkbcomp\" after variable name \"${XKBBASE}\".",
		"WARN: x11/xkeyboard-config/Makefile:3: "+
			"Invalid part \"/xkbcomp\" after variable name \"${XKBBASE}\".",
		"WARN: x11/xkeyboard-config/Makefile:3: Variable \"XKBBASE\" is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_checkEmptyContinuation(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("filename.mk",
		MkCvsID,
		"# line 1 \\",
		"",
		"# line 2")

	// Don't check this when loading a file, since otherwise the infrastructure
	// files could possibly get this warning. Sure, they should be fixed, but
	// it's not in the focus of the package maintainer.
	t.CheckOutputEmpty()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: ~/filename.mk:3: This line looks empty but continues the previous line.")
}

func (s *Suite) Test_MkLineChecker_checkTextExpr(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"VAR=\t${:U",
		".info ${VAR}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: Missing closing \"}\" for \"\".")
}

func (s *Suite) Test_MkLineChecker_checkText(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()

	mklines := t.SetUpFileMkLines("module.mk",
		MkCvsID,
		"CFLAGS+=\t\t-Wl,--rpath,${PREFIX}/lib",
		"PKG_FAIL_REASON+=\t\"Group ${GAMEGRP} doesn't exist.\"")
	t.FinishSetUp()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: ~/module.mk:2: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,--rpath,\".",
		"WARN: ~/module.mk:3: Use of \"GAMEGRP\" is deprecated. Use GAMES_GROUP instead.")
}

func (s *Suite) Test_MkLineChecker_checkText__WRKSRC(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--explain")
	mklines := t.SetUpFileMkLines("module.mk",
		MkCvsID,
		"pre-configure:",
		"\tcd ${WRKSRC}/..")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: ~/module.mk:3: Building the package should take place entirely inside ${WRKSRC}, not \"${WRKSRC}/..\".",
		"",
		"\tWRKSRC should be defined so that there is no need to do anything",
		"\toutside of this directory.",
		"",
		"\tExample:",
		"",
		"\t\tWRKSRC=\t\t${WRKDIR}",
		"\t\tCONFIGURE_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src",
		"\t\tBUILD_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src ${WRKSRC}/cmd",
		"",
		"\tSee the pkgsrc guide, section \"Directories used during the build",
		"\tprocess\":",
		"\thttps://www.NetBSD.org/docs/pkgsrc/pkgsrc.html#build.builddirs",
		"",
		"WARN: ~/module.mk:3: Variable \"WRKSRC\" is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_checkTextWrksrcDotDot(c *check.C) {
	t := s.Init(c)
	t.SetUpVartypes()
	t.SetUpTool("echo", "ECHO", AfterPrefsMk)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"do-build:",
		"\techo ${WRKSRC}/..")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: " +
			"Building the package should take place entirely " +
			"inside ${WRKSRC}, not \"${WRKSRC}/..\".")
}

// In general, -Wl,-R should not appear in package Makefiles.
// BUILDLINK_TRANSFORM is an exception to this since this command line option
// is removed here from the compiler invocations.
func (s *Suite) Test_MkLineChecker_checkTextRpath(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"BUILDLINK_TRANSFORM+=\t\trm:-Wl,-R/usr/lib",
		"BUILDLINK_TRANSFORM+=\t\trm:-Wl,-rpath,/usr/lib",
		"BUILDLINK_TRANSFORM+=\t\topt:-Wl,-rpath,/usr/lib",
		"BUILDLINK_TRANSFORM.pkgbase+=\trm:-Wl,-R/usr/lib",
		"BUILDLINK_TRANSFORM.pkgbase+=\trm:-Wl,-rpath,/usr/lib",
		"BUILDLINK_TRANSFORM.pkgbase+=\topt:-Wl,-rpath,/usr/lib",
		"",
		"LDFLAGS+=\t-Wl,-R${PREFIX}/gcc9/lib",
		"LDFLAGS+=\t-Wl,-rpath,${PREFIX}/gcc9/lib",
		"LDFLAGS+=\t-Wl,--rpath,${PREFIX}/gcc9/lib")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:4: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-rpath,\".",
		"WARN: filename.mk:7: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-rpath,\".",
		// TODO: Remove duplicates.
		"WARN: filename.mk:9: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-R\".",
		"WARN: filename.mk:9: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-R\".",
		// TODO: Remove duplicates.
		"WARN: filename.mk:10: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-rpath,\".",
		"WARN: filename.mk:10: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,-rpath,\".",
		// TODO: Remove duplicates.
		"WARN: filename.mk:11: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,--rpath,\".",
		"WARN: filename.mk:11: Use ${COMPILER_RPATH_FLAG} instead of \"-Wl,--rpath,\".")
}

func (s *Suite) Test_MkLineChecker_checkTextMissingDollar(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"CONFIGURE_ARGS+=\t--with-fltk={BUILDLINK_DIR:Q}/nonexistent",
		"INDIRECT=\t{:Ufallback}",
		"PRINT_PLIST_AWK+=\t{sub(\"^.*em:id=\\\"\", \"\");sub(\"\\\".*$$\",\"\");print $$0}")

	for _, mkline := range mklines.mklines {
		if mkline.IsVarassign() {
			ck := NewMkLineChecker(mklines, mkline)
			ck.checkText(mkline.Value())
		}
	}

	t.CheckOutputLines(
		"WARN: filename.mk:2: "+
			"Maybe missing '$' in expression \"{BUILDLINK_DIR:Q}\".",
		"WARN: filename.mk:3: "+
			"Maybe missing '$' in expression \"{:Ufallback}\".")
}

func (s *Suite) Test_MkLineChecker_checkVartype__simple_type(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	// Since COMMENT is defined in vardefs.go its type is certain instead of guessed.
	vartype := G.Pkgsrc.VariableType(nil, "COMMENT")

	t.AssertNotNil(vartype)
	t.CheckEquals(vartype.basicType.name, "Comment")
	t.CheckEquals(vartype.IsGuessed(), false)
	t.CheckEquals(vartype.IsList(), no)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"COMMENT=\tA nice package")
	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:2: COMMENT should not begin with \"A\".")
}

func (s *Suite) Test_MkLineChecker_checkVartype(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"DISTNAME=\tgcc-${GCC_VERSION}")

	mklines.allVars.Define("GCC_VERSION", mklines.mklines[1])
	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkVartype__append_to_non_list(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"DISTNAME+=\tsuffix",
		"COMMENT=\tComment for",
		"COMMENT+=\tthe package",
		"",
		"GITHUB_SUBMODULES+=\tmodule1 module2",
	)

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: The variable DISTNAME should not be appended to "+
			"(only set, or given a default value) in this file.",
		"WARN: filename.mk:2: The \"+=\" operator should only be used with lists, not with DISTNAME.",
		"WARN: filename.mk:6: Appending to GITHUB_SUBMODULES "+
			"should happen in groups of 4 words each, not 2.",
	)
}

func (s *Suite) Test_MkLineChecker_checkVartype__no_tracing(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"UNKNOWN=\tvalue",
		"CUR_DIR!=\tpwd")
	t.DisableTracing()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: Variable \"UNKNOWN\" is defined but not used.",
		"WARN: filename.mk:3: Variable \"CUR_DIR\" is defined but not used.")
}

func (s *Suite) Test_MkLineChecker_checkVartype__one_per_line(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"PKG_FAIL_REASON+=\tSeveral words are wrong.",
		"PKG_FAIL_REASON+=\t\"Properly quoted\"",
		"PKG_FAIL_REASON+=\t# none")
	t.DisableTracing()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: PKG_FAIL_REASON should only get one item per line.")
}

func (s *Suite) Test_MkLineChecker_checkVartype__CFLAGS_with_backticks(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("chat/pidgin-icb/Makefile",
		MkCvsID,
		"CFLAGS+=\t`pkg-config pidgin --cflags`")
	mkline := mklines.mklines[1]

	words := mkline.Fields()

	// bmake handles backticks in the same way, treating them as ordinary characters
	t.CheckDeepEquals(words, []string{"`pkg-config", "pidgin", "--cflags`"})

	ck := MkLineChecker{mklines, mklines.mklines[1]}
	ck.checkVartype("CFLAGS", opAssignAppend, "`pkg-config pidgin --cflags`", "")

	// No warning about "`pkg-config" being an unknown CFlag.
	// As of September 2019, there is no such check anymore in pkglint.
	t.CheckOutputEmpty()
}

// See PR 46570, Ctrl+F "4. Shell quoting".
// Pkglint is correct, since the shell sees this definition for
// CPPFLAGS as three words, not one word.
func (s *Suite) Test_MkLineChecker_checkVartype__CFLAGS(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"CPPFLAGS.SunOS+=\t-DPIPECOMMAND=\\\"/usr/sbin/sendmail -bs %s\\\"")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:2: Compiler flag \"-DPIPECOMMAND=\\\\\\\"/usr/sbin/sendmail\" has unbalanced double quotes.",
		"WARN: Makefile:2: Compiler flag \"%s\\\\\\\"\" has unbalanced double quotes.")
}

func (s *Suite) Test_MkLineChecker_CheckVartypeBasic(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"PKGREVISION=\tnone")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: "+
			"The variable PKGREVISION should not be set "+
			"in this file; it would be ok in Makefile.",
		"ERROR: filename.mk:2: "+
			"PKGREVISION must be a positive integer number.",
		"ERROR: filename.mk:2: PKGREVISION only makes sense "+
			"directly in the package Makefile.")
}

func (s *Suite) Test_MkLineChecker_checkShellCommand__indentation(c *check.C) {
	t := s.Init(c)

	doTest := func(bool) {
		mklines := t.SetUpFileMkLines("filename.mk",
			MkCvsID,
			"",
			"do-install:",
			"\t\techo 'unnecessarily indented'",
			"\t\tfor var in 1 2 3; do \\",
			"\t\t\techo \"$$var\"; \\",
			"\t                echo \"spaces\"; \\",
			"\t\tdone",
			"",
			"\t\t\t\t\t# comment, not a shell command")

		mklines.Check()
	}

	t.ExpectDiagnosticsAutofix(
		doTest,
		"NOTE: ~/filename.mk:4: Shell programs should be indented with a single tab.",
		"WARN: ~/filename.mk:4: Unknown shell command \"echo\".",
		"NOTE: ~/filename.mk:5--8: Shell programs should be indented with a single tab.",
		"WARN: ~/filename.mk:5--8: Unknown shell command \"echo\".",
		"WARN: ~/filename.mk:5--8: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"echo \\\"$$var\\\"\") to separate commands.",
		"WARN: ~/filename.mk:5--8: Unknown shell command \"echo\".",

		"AUTOFIX: ~/filename.mk:4: Replacing \"\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/filename.mk:5: Replacing \"\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/filename.mk:6: Replacing \"\\t\\t\" with \"\\t\".",
		"AUTOFIX: ~/filename.mk:8: Replacing \"\\t\\t\" with \"\\t\".")
	t.CheckFileLinesDetab("filename.mk",
		MkCvsID,
		"",
		"do-install:",
		"        echo 'unnecessarily indented'",
		"        for var in 1 2 3; do \\",
		"                echo \"$$var\"; \\",
		"                        echo \"spaces\"; \\", // not changed
		"        done",
		"",
		"                                        # comment, not a shell command")
}

func (s *Suite) Test_MkLineChecker_checkComment(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"# url2pkg-marker")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: filename.mk:2: This comment indicates unfinished work (url2pkg).")
}

func (s *Suite) Test_MkLineChecker_checkInclude(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	t.CreateFileLines("pkgtools/x11-links/buildlink3.mk")
	t.CreateFileLines("graphics/jpeg/buildlink3.mk")
	t.CreateFileLines("devel/intltool/buildlink3.mk")
	t.CreateFileLines("devel/intltool/builtin.mk")
	t.CreateFileLines("mk/bsd.pkg.mk")
	mklines := t.SetUpFileMkLines("category/package/filename.mk",
		MkCvsID,
		"",
		".include \"../../pkgtools/x11-links/buildlink3.mk\"",
		".include \"../../graphics/jpeg/buildlink3.mk\"",
		".include \"../../devel/intltool/buildlink3.mk\"",
		".include \"../../devel/intltool/builtin.mk\"",
		".include \"/absolute\"",
		".include \"../../mk/bsd.pkg.mk\"")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: ~/category/package/filename.mk:7: "+
			"Unknown makefile line format: \".include \\\"/absolute\\\"\".",
		"ERROR: ~/category/package/filename.mk:3: "+
			"\"../../pkgtools/x11-links/buildlink3.mk\" must not be included directly. "+
			"Include \"../../mk/x11.buildlink3.mk\" instead.",
		"ERROR: ~/category/package/filename.mk:4: "+
			"\"../../graphics/jpeg/buildlink3.mk\" must not be included directly. "+
			"Include \"../../mk/jpeg.buildlink3.mk\" instead.",
		"WARN: ~/category/package/filename.mk:5: "+
			"Write \"USE_TOOLS+= intltool\" instead of this line.",
		"ERROR: ~/category/package/filename.mk:6: "+
			"\"../../devel/intltool/builtin.mk\" must not be included directly. "+
			"Include \"../../devel/intltool/buildlink3.mk\" instead.",
		"ERROR: ~/category/package/filename.mk:8: "+
			"The file bsd.pkg.mk must only be included by package Makefiles, "+
			"not by other makefile fragments.")
}

func (s *Suite) Test_MkLineChecker_checkInclude__Makefile(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines(t.File("Makefile"),
		MkCvsID,
		".include \"../../other/package/Makefile\"")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: ~/Makefile:2: Relative path \"../../other/package/Makefile\" does not exist.",
		"ERROR: ~/Makefile:2: Other Makefiles must not be included directly.")
}

func (s *Suite) Test_MkLineChecker_checkInclude__Makefile_exists(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("other/existing/Makefile")
	t.SetUpPackage("category/package",
		".include \"../../other/existing/Makefile\"",
		".include \"../../other/not-found/Makefile\"")
	t.FinishSetUp()

	G.checkdirPackage(t.File("category/package"))

	t.CheckOutputLines(
		"ERROR: ~/category/package/Makefile:21: Cannot read \"../../other/not-found/Makefile\".")
}

func (s *Suite) Test_MkLineChecker_checkInclude__hacks(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	t.CreateFileLines("category/package/hacks.mk",
		MkCvsID,
		".include \"../../category/package/nonexistent.mk\"",
		".include \"../../category/package/builtin.mk\"")
	t.CreateFileLines("category/package/builtin.mk",
		MkCvsID)
	t.FinishSetUp()

	G.checkdirPackage(t.File("category/package"))

	// The purpose of this "nonexistent" diagnostic is only to show that
	// hacks.mk is indeed parsed and checked.
	t.CheckOutputLines(
		// XXX: The diagnostics should be the same since they
		//  describe the same problem.
		"ERROR: ~/category/package/hacks.mk:2: "+
			"Cannot read \"../../category/package/nonexistent.mk\".",
		"ERROR: ~/category/package/hacks.mk:2: "+
			"Relative path \"../../category/package/nonexistent.mk\" does not exist.")
}

func (s *Suite) Test_MkLineChecker_checkIncludePythonWheel__no_restrictions(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("lang/python/egg.mk",
		MkCvsID)
	t.SetUpPackage("devel/py-test")
	t.FinishSetUp()
	t.Chdir("devel/py-test")

	G.Check(".")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkIncludePythonWheel__not_Python_2(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("lang/python/egg.mk",
		MkCvsID)
	t.SetUpPackage("devel/py-test",
		"PYTHON_VERSIONS_INCOMPATIBLE=\t27",
		".include \"../../lang/python/egg.mk\"")
	t.FinishSetUp()
	t.Chdir("devel/py-test")

	G.Check(".")

	t.CheckOutputLines(
		"WARN: Makefile:21: Python egg.mk is deprecated, use wheel.mk instead.")
}

func (s *Suite) Test_MkLineChecker_checkIncludePythonWheel__not_Python_3(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("lang/python/egg.mk",
		MkCvsID)
	t.SetUpPackage("devel/py-test",
		"PYTHON_VERSIONS_INCOMPATIBLE=\t38\t# rationale",
		".include \"../../lang/python/egg.mk\"")
	t.FinishSetUp()
	t.Chdir("devel/py-test")

	G.Check(".")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkIncludePythonWheel__only_Python_2(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("lang/python/egg.mk",
		MkCvsID)
	t.SetUpPackage("devel/py-test",
		"PYTHON_VERSIONS_ACCEPTED=\t27\t# rationale",
		".include \"../../lang/python/egg.mk\"")
	t.FinishSetUp()
	t.Chdir("devel/py-test")

	G.Check(".")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkIncludePythonWheel__only_Python_3(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("lang/python/egg.mk",
		MkCvsID)
	t.SetUpPackage("devel/py-test",
		"PYTHON_VERSIONS_ACCEPTED=\t310 38\t# rationale",
		".include \"../../lang/python/egg.mk\"")
	t.FinishSetUp()
	t.Chdir("devel/py-test")

	G.Check(".")

	t.CheckOutputLines(
		"WARN: Makefile:21: Python egg.mk is deprecated, use wheel.mk instead.")
}

// A buildlink3.mk file may include its corresponding builtin.mk file directly.
func (s *Suite) Test_MkLineChecker_checkIncludeBuiltin__buildlink3_mk(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	mklines := t.SetUpFileMkLines("category/package/buildlink3.mk",
		MkCvsID,
		".include \"builtin.mk\"")
	t.CreateFileLines("category/package/builtin.mk",
		MkCvsID)
	t.FinishSetUp()

	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkIncludeBuiltin__rationale(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"# I have good reasons for including this file directly.",
		".include \"../../category/package/builtin.mk\"",
		"",
		".include \"../../category/package/builtin.mk\"",
		".include \"../../category/package/builtin.mk\" # intentionally included directly")
	t.CreateFileLines("category/package/builtin.mk",
		MkCvsID)
	t.FinishSetUp()

	G.checkdirPackage(t.File("category/package"))

	t.CheckOutputLines(
		"ERROR: ~/category/package/Makefile:23: " +
			"\"../../category/package/builtin.mk\" must not be included directly. " +
			"Include \"../../category/package/buildlink3.mk\" instead.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveIndentation__autofix(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("--autofix")
	lines := t.SetUpFileLines("filename.mk",
		MkCvsID,
		".if defined(A)",
		".for a in ${A}",
		".if defined(C)",
		".endif",
		".endfor",
		".endif")
	mklines := NewMkLines(lines, nil, nil)

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/filename.mk:3: Replacing \".\" with \".  \".",
		"AUTOFIX: ~/filename.mk:4: Replacing \".\" with \".    \".",
		"AUTOFIX: ~/filename.mk:5: Replacing \".\" with \".    \".",
		"AUTOFIX: ~/filename.mk:6: Replacing \".\" with \".  \".")
	t.CheckFileLines("filename.mk",
		"# $"+"NetBSD$",
		".if defined(A)",
		".  for a in ${A}",
		".    if defined(C)",
		".    endif",
		".  endfor",
		".endif")
}

// Up to 2018-01-28, pkglint applied the autofix also to the continuation
// lines, which is incorrect. It replaced the dot in "4.*" with spaces.
func (s *Suite) Test_MkLineChecker_checkDirectiveIndentation__autofix_multiline(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	t.SetUpVartypes()
	mklines := t.SetUpFileMkLines("options.mk",
		MkCvsID,
		".if ${PKGNAME} == pkgname",
		".if \\",
		"   ${PLATFORM:MNetBSD-4.*}",
		".endif",
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: ~/options.mk:3: Replacing \".\" with \".  \".",
		"AUTOFIX: ~/options.mk:5: Replacing \".\" with \".  \".")

	t.CheckFileLines("options.mk",
		MkCvsID,
		".if ${PKGNAME} == pkgname",
		".  if \\",
		"   ${PLATFORM:MNetBSD-4.*}",
		".  endif",
		".endif")
}

// Having a continuation line between the dot and the directive is so
// unusual that pkglint doesn't fix it automatically. It also doesn't panic.
func (s *Suite) Test_MkLineChecker_checkDirectiveIndentation__multiline(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	t.ExpectDiagnosticsAutofix(
		func(autofix bool) {
			mklines := t.SetUpFileMkLines("options.mk",
				MkCvsID,
				".\\",
				"if ${MACHINE_PLATFORM:MNetBSD-4.*}",
				".endif")

			mklines.Check()
		},
		"NOTE: ~/options.mk:2--3: "+
			"This directive should be indented by 0 spaces.",
		"WARN: ~/options.mk:2--3: "+
			"To use MACHINE_PLATFORM at load time, "+
			".include \"mk/bsd.prefs.mk\" first.")
}

// Another strange edge case that doesn't occur in practice.
func (s *Suite) Test_MkLineChecker_checkDirectiveIndentation__multiline_indented(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	doTest := func(autofix bool) {
		mklines := t.SetUpFileMkLines("options.mk",
			MkCvsID,
			". \\",
			"if ${PLATFORM:MNetBSD-4.*}",
			".endif")

		mklines.Check()
	}

	t.ExpectDiagnosticsAutofix(
		doTest,
		"NOTE: ~/options.mk:2: This directive should be indented by 0 spaces.",
		"WARN: ~/options.mk:2--3: Variable \"PLATFORM\" is used but not defined.",
		// If the indentation should ever change here, it is probably
		// because MkLineParser.parseDirective has been changed to
		// behave more like bmake, which preserves a bit more of the
		// whitespace.
		"AUTOFIX: ~/options.mk:2: Replacing \". \" with \".\".")

	// It's not really fixed since the backslash is still replaced
	// with a single space when being parsed.
	// At least pkglint doesn't make the situation worse than before.
	t.CheckFileLines("options.mk",
		MkCvsID,
		".\\",
		"if ${PLATFORM:MNetBSD-4.*}",
		".endif")
}

func (s *Suite) Test_MkLineChecker_CheckRelativePath(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	t.CreateFileLines("wip/package/Makefile")
	t.CreateFileLines("wip/package/module.mk")
	mklines := t.SetUpFileMkLines("category/package/module.mk",
		MkCvsID,
		"DEPENDS+=       wip-package-[0-9]*:../../wip/package",
		".include \"../../wip/package/module.mk\"",
		"",
		"DEPENDS+=       unresolvable-[0-9]*:../../lang/${LATEST_PYTHON}",
		".include \"../../lang/${LATEST_PYTHON}/module.mk\"",
		"",
		".include \"module.mk\"",
		".include \"../../category/../category/package/module.mk\"", // Oops
		".include \"../../mk/bsd.prefs.mk\"",
		".include \"../package/module.mk\"",
		// TODO: warn about this as well, since ${.CURDIR} is essentially
		//  equivalent to ".".
		".include \"${.CURDIR}/../package/module.mk\"")
	t.FinishSetUp()

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: ~/category/package/module.mk:2: A main pkgsrc package must not depend on a pkgsrc-wip package.",
		"ERROR: ~/category/package/module.mk:3: A main pkgsrc package must not depend on a pkgsrc-wip package.",
		"WARN: ~/category/package/module.mk:5: Variable \"LATEST_PYTHON\" is used but not defined.",
		"WARN: ~/category/package/module.mk:11: References to other packages should "+
			"look like \"../../category/package\", not \"../package\".",
		"WARN: ~/category/package/module.mk:12: References to other packages should "+
			"look like \"../../category/package\", not \"../package\".")
}

func (s *Suite) Test_MkLineChecker_CheckRelativePath__absolute_path(c *check.C) {
	t := s.Init(c)

	absDir := condStr(runtime.GOOS == "windows", "C:/", "/")
	// Just a random UUID, to really guarantee that the file does not exist.
	absPath := absDir + "0f5c2d56-8a7a-4c9d-9caa-859b52bbc8c7"

	t.SetUpPkgsrc()
	mklines := t.SetUpFileMkLines("category/package/module.mk",
		MkCvsID,
		"DISTINFO_FILE=\t"+absPath)
	t.FinishSetUp()

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: ~/category/package/module.mk:2: The path \"" + absPath + "\" must be relative.")
}

func (s *Suite) Test_MkLineChecker_CheckRelativePath__include_if_exists(c *check.C) {
	t := s.Init(c)

	mklines := t.SetUpFileMkLines("filename.mk",
		MkCvsID,
		".include \"included.mk\"",
		".sinclude \"included.mk\"")

	mklines.Check()

	// There is no warning for line 3 because of the "s" in "sinclude".
	t.CheckOutputLines(
		"ERROR: ~/filename.mk:2: Relative path \"included.mk\" does not exist.")
}

func (s *Suite) Test_MkLineChecker_CheckRelativePath__wip_mk(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("wip/mk/git-package.mk",
		MkCvsID)
	t.CreateFileLines("wip/other/version.mk",
		MkCvsID)
	t.SetUpPackage("wip/package",
		".include \"../mk/git-package.mk\"",
		".include \"../other/version.mk\"")
	t.FinishSetUp()

	G.Check(t.File("wip/package"))

	t.CheckOutputLines(
		"WARN: ~/wip/package/Makefile:20: References to the pkgsrc-wip "+
			"infrastructure should look like \"../../wip/mk\", not \"../mk\".",
		"WARN: ~/wip/package/Makefile:21: References to other packages "+
			"should look like \"../../category/package\", not \"../package\".",
		"WARN: ~/wip/package/COMMIT_MSG: Every work-in-progress "+
			"package should have a COMMIT_MSG file.")
}

func (s *Suite) Test_MkLineChecker_CheckPackageDir(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("other/package/Makefile")

	test := func(packagePath PackagePath, diagnostics ...string) {
		// Must be in the filesystem because of directory references.
		mklines := t.SetUpFileMkLines("category/package/Makefile",
			"# dummy")

		checkPackageDir := func(mkline *MkLine) {
			ck := MkLineChecker{mklines, mkline}
			ck.CheckPackageDir(packagePath)
		}

		mklines.ForEach(checkPackageDir)

		t.CheckOutput(diagnostics)
	}

	test("../pkgbase",
		"ERROR: ~/category/package/Makefile:1: Relative path \"../pkgbase/Makefile\" does not exist.",
		"WARN: ~/category/package/Makefile:1: \"../pkgbase\" is not a valid relative package directory.")

	test("../../other/package",
		nil...)

	test("../../other/does-not-exist",
		"ERROR: ~/category/package/Makefile:1: Relative path \"../../other/does-not-exist/Makefile\" does not exist.")

	test("${OTHER_PACKAGE}",
		nil...)
}

func (s *Suite) Test_MkLineChecker_checkDirective(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("category/package/filename.mk",
		MkCvsID,
		"",
		".for",
		".endfor",
		"",
		".if",
		".else don't",
		".endif invalid-arg",
		"",
		".ifdef FNAME_MK",
		".endif",
		".ifndef FNAME_MK",
		".endif",
		"",
		".for var in a b c",
		".endfor",
		".undef var unrelated",
		"",
		".if 0",
		".  info Unsupported operating system",
		".  warning Unsupported operating system",
		".  error Unsupported operating system",
		".  export-env A",
		".  unexport A",
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: category/package/filename.mk:3: \".for\" requires arguments.",
		"ERROR: category/package/filename.mk:6: \".if\" requires arguments.",
		"ERROR: category/package/filename.mk:7: \".else\" does not take arguments. "+
			"If you meant \"else if\", use \".elif\".",
		"ERROR: category/package/filename.mk:8: \".endif\" does not take arguments.",
		"WARN: category/package/filename.mk:10: The \".ifdef\" directive is deprecated. "+
			"Use \".if defined(FNAME_MK)\" instead.",
		"WARN: category/package/filename.mk:12: The \".ifndef\" directive is deprecated. "+
			"Use \".if !defined(FNAME_MK)\" instead.",
		"NOTE: category/package/filename.mk:17: Using \".undef\" after a \".for\" loop is unnecessary.")
}

func (s *Suite) Test_MkLineChecker_checkDirective__for_loop_varname(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		".for VAR in a b c", // Should be lowercase.
		".endfor",
		"",
		".for _var_ in a b c", // Should be written without underscores.
		".endfor",
		"",
		".for .var. in a b c", // Should be written without dots.
		".endfor",
		"",
		".for ${VAR} in a b c", // The variable name really must be an identifier.
		".endfor")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: The variable name \"VAR\" in the .for loop should not contain uppercase letters.",
		"WARN: filename.mk:6: Variable names starting with an underscore (_var_) are reserved for internal pkgsrc use.",
		"ERROR: filename.mk:9: Invalid variable name \".var.\".",
		"ERROR: filename.mk:12: Invalid variable name \"${VAR}\".")
}

// TODO: Split into separate tests.
func (s *Suite) Test_MkLineChecker_checkDirectiveEnd__ending_comments(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	t.Chdir("category/package")
	t.FinishSetUp()
	mklines := t.NewMkLines("opsys.mk",
		MkCvsID,
		"",
		".include \"../../mk/bsd.prefs.mk\"",
		"",
		".for i in 1 2 3 4 5",
		".  if ${OPSYS} == NetBSD",
		".    if ${MACHINE_ARCH} == x86_64",
		".      if ${OS_VERSION:M8.*}",
		".      endif # MACHINE_ARCH", // Wrong, should be OS_VERSION.
		".    endif # OS_VERSION",     // Wrong, should be MACHINE_ARCH.
		".  endif # OPSYS",            // Correct.
		".endfor # j",                 // Wrong, should be i.
		"",
		".if ${PKG_OPTIONS:Moption}",
		".endif # option", // Correct.
		"",
		".if ${PKG_OPTIONS:Moption}",
		".endif # opti", // This typo goes unnoticed since "opti" is a substring of the condition.
		"",
		".if ${OPSYS} == NetBSD",
		".elif ${OPSYS} == FreeBSD",
		".endif # NetBSD", // Wrong, should be OPSYS, which applies to all branches.
		"",                // Or FreeBSD since that is the branch being closed right now.
		".for ii in 1 2",
		".  for jj in 1 2",
		".  endfor # ii", // Note: a simple "i" would not generate a warning because it is found in the word "in".
		".endfor # ii")

	// See MkLineChecker.checkDirective
	mklines.Check()

	t.CheckOutputLines(
		"WARN: opsys.mk:9: Comment \"MACHINE_ARCH\" does not match condition \"${OS_VERSION:M8.*}\" in line 8.",
		"WARN: opsys.mk:10: Comment \"OS_VERSION\" does not match condition \"${MACHINE_ARCH} == x86_64\" in line 7.",
		"WARN: opsys.mk:12: Comment \"j\" does not match loop \"i in 1 2 3 4 5\" in line 5.",
		"WARN: opsys.mk:14: Undocumented option \"option\".",
		"WARN: opsys.mk:22: Comment \"NetBSD\" does not match condition \"${OPSYS} == FreeBSD\" in line 21.",
		"WARN: opsys.mk:26: Comment \"ii\" does not match loop \"jj in 1 2\" in line 25.")
}

// After removing the dummy indentation in commit d5a926af,
// there was a panic: runtime error: index out of range,
// in wip/jacorb-lib/buildlink3.mk.
func (s *Suite) Test_MkLineChecker_checkDirectiveEnd__unbalanced(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		".endfor # comment",
		".endif # comment")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: filename.mk:3: Unmatched .endfor.",
		"ERROR: filename.mk:4: Unmatched .endif.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveEnd__ifmake(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		".ifmake links",
		".endif")

	mklines.Check()

	// FIXME: Either warn about ".ifmake" or don't error here.
	t.CheckOutputLines(
		"ERROR: Makefile:3: Unmatched .endif.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveFor(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("for.mk",
		MkCvsID,
		".for dir in ${PATH:C,:, ,g}",
		".endfor",
		"",
		".for dir in ${PATH}",
		".endfor",
		"",
		".for dir in ${PATH:M*/bin}",
		".endfor",
		"",
		".for opt in $(PATH)", // Parentheses instead of braces.
		".endfor",
		"",
		"DISTFILES=\t# none",
		".for gem in ${DISTFILES:M*.gem:S/.gem$//g}",
		".endfor")

	mklines.Check()

	t.CheckOutputLines(
		// No warning about a missing :Q in line 2 since the :C modifier
		// converts the colon-separated list into a space-separated list,
		// as required by the .for loop.

		// FIXME: Do not suggest the ':Q' modifier, instead suggest to use spaces.
		"WARN: for.mk:5: Use ${PATH:Q} instead of ${PATH}.",

		// Applying the ':M' modifier with a pattern to a single-value
		// variable can be useful, as a form of pattern matching.
		// More probably though, the intention was to see whether the PATH
		// has _any_ directory ending in "/bin", instead of testing only
		// the last directory.
		// FIXME: Do not suggest the ':Q' modifier, instead suggest to use spaces.
		"WARN: for.mk:8: Use ${PATH:M*/bin:Q} instead of ${PATH:M*/bin}.",

		// TODO: Warn about round parentheses instead of curly braces.
		// FIXME: Do not suggest the ':Q' modifier, instead suggest to use spaces.
		"WARN: for.mk:11: Use ${PATH:Q} instead of ${PATH}.",

		// TODO: Why not? The variable is guaranteed to be defined at this point.
		"WARN: for.mk:15: DISTFILES should not be used at load time in any file.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveFor__continuation(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"do-install:",
		".for d in \\",
		"    dir1 dir2 \\",
		"    dir3 dir4 \\",
		"    dir5 dir6 \\", // This line continuation is not intended
		"\t: mkdir $d",
		// This .for loop has an empty body.
		".endfor")

	mklines.Check()

	// TODO: Warn about the unintended line continuation.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkDirectiveFor__items(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".for f in $< $a $0",
		".endfor")

	mklines.Check()

	// TODO: Warn about $a and $0 being ambiguous.
	t.CheckOutputLines(
		"WARN: filename.mk:2: Variable \"<\" is used but not defined.",
		"WARN: filename.mk:2: Variable \"a\" is used but not defined.",
		"WARN: filename.mk:2: Variable \"0\" is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveFor__infrastructure(c *check.C) {
	t := s.Init(c)

	t.SetUpPkgsrc()
	t.CreateFileLines("mk/file.mk",
		MkCvsID,
		".for i = 1 2 3", // The "=" should rather be "in".
		".endfor",
		"",
		".for _i_ in 1 2 3", // Underscores are only allowed in infrastructure files.
		".endfor")
	t.FinishSetUp()

	G.Check(t.File("mk/file.mk"))

	// Pkglint doesn't care about trivial syntax errors like the "=" instead
	// of "in" above; bmake will already catch these.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkDependencyRule(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("category/package/filename.mk",
		MkCvsID,
		"",
		".PHONY: target-1",
		"target-2: .PHONY",
		".ORDER: target-1 target-2",
		"target-1:",
		"target-2:",
		"target-3:",
		"${_COOKIE.test}:")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: category/package/filename.mk:8: Undeclared target \"target-3\".")
}

func (s *Suite) Test_MkLineChecker_checkDependencyTarget(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"unknown-target:",
		"\t:;")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:3: Undeclared target \"unknown-target\".",
		"NOTE: filename.mk:4: A trailing semicolon "+
			"at the end of a shell command line is redundant.")
}
