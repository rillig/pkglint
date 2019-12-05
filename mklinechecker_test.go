package pkglint

import (
	"gopkg.in/check.v1"
	"runtime"
)

// PR pkg/46570, item 2
func (s *Suite) Test_MkLineChecker__unclosed_varuse(c *check.C) {
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
		"WARN: Makefile:2: EGDIRS is defined but not used.",
		"WARN: Makefile:2: EGDIR/pam.d is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_Check__url2pkg(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"# url2pkg-marker")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: filename.mk:2: This comment indicates unfinished work (url2pkg).")
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

	// If the buildlink3.mk file doesn't actually exist, resolving the
	// relative path fails since that depends on the actual file system,
	// not on syntactical paths; see os.Stat in CheckRelativePath.
	//
	// TODO: Refactor Relpath to be independent of a filesystem.

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/category/package/buildlink3.mk:2: For efficiency reasons, " +
			"please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
}

func (s *Suite) Test_MkLineChecker_Check__warn_varuse_LOCALBASE(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("options.mk",
		MkCvsID,
		"PKGNAME=\t${LOCALBASE}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: options.mk:2: Please use PREFIX instead of LOCALBASE.")
}

func (s *Suite) Test_MkLineChecker_Check__varuse_modifier_L(c *check.C) {
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
	// In line 3 the :L modifier is missing, therefore ${XKBBASE}/xkbcomp is the
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
		"WARN: x11/xkeyboard-config/Makefile:3: XKBBASE is used but not defined.")
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
		"NOTE: ~/filename.mk:2--3: Trailing whitespace.",
		"WARN: ~/filename.mk:3: This line looks empty but continues the previous line.")
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
		"WARN: ~/module.mk:2: Please use ${COMPILER_RPATH_FLAG} instead of \"-Wl,--rpath,\".",
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
		"\t\tWRKSRC=\t${WRKDIR}",
		"\t\tCONFIGURE_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src",
		"\t\tBUILD_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src ${WRKSRC}/cmd",
		"",
		"\tSee the pkgsrc guide, section \"Directories used during the build",
		"\tprocess\":",
		"\thttps://www.NetBSD.org/docs/pkgsrc/pkgsrc.html#build.builddirs",
		"",
		"WARN: ~/module.mk:3: WRKSRC is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_checkVartype__simple_type(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	// Since COMMENT is defined in vardefs.go its type is certain instead of guessed.
	vartype := G.Pkgsrc.VariableType(nil, "COMMENT")

	c.Assert(vartype, check.NotNil)
	t.CheckEquals(vartype.basicType.name, "Comment")
	t.CheckEquals(vartype.IsGuessed(), false)
	t.CheckEquals(vartype.IsList(), false)

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

	mklines.vars.Define("GCC_VERSION", mklines.mklines[1])
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
		"COMMENT+=\tthe package")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:2: The variable DISTNAME should not be appended to "+
			"(only set, or given a default value) in this file.",
		"WARN: filename.mk:2: The \"+=\" operator should only be used with lists, not with DISTNAME.")
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
		"WARN: filename.mk:2: UNKNOWN is defined but not used.",
		"WARN: filename.mk:3: CUR_DIR is defined but not used.")
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

func (s *Suite) Test_MkLineChecker_checkShellCommand__indentation(c *check.C) {
	t := s.Init(c)

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
	t.SetUpCommandLine("-Wall", "--autofix")
	mklines.Check()

	t.CheckOutputLines(
		"NOTE: ~/filename.mk:4: Shell programs should be indented with a single tab.",
		"WARN: ~/filename.mk:4: Unknown shell command \"echo\".",
		"NOTE: ~/filename.mk:5--8: Shell programs should be indented with a single tab.",
		"WARN: ~/filename.mk:5--8: Unknown shell command \"echo\".",
		"WARN: ~/filename.mk:5--8: Please switch to \"set -e\" mode before using a semicolon "+
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

func (s *Suite) Test_MkLineChecker_checkInclude(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	t.CreateFileLines("pkgtools/x11-links/buildlink3.mk")
	t.CreateFileLines("graphics/jpeg/buildlink3.mk")
	t.CreateFileLines("devel/intltool/buildlink3.mk")
	t.CreateFileLines("devel/intltool/builtin.mk")
	mklines := t.SetUpFileMkLines("category/package/filename.mk",
		MkCvsID,
		"",
		".include \"../../pkgtools/x11-links/buildlink3.mk\"",
		".include \"../../graphics/jpeg/buildlink3.mk\"",
		".include \"../../devel/intltool/buildlink3.mk\"",
		".include \"../../devel/intltool/builtin.mk\"")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: ~/category/package/filename.mk:3: "+
			"\"../../pkgtools/x11-links/buildlink3.mk\" must not be included directly. "+
			"Include \"../../mk/x11.buildlink3.mk\" instead.",
		"ERROR: ~/category/package/filename.mk:4: "+
			"\"../../graphics/jpeg/buildlink3.mk\" must not be included directly. "+
			"Include \"../../mk/jpeg.buildlink3.mk\" instead.",
		"WARN: ~/category/package/filename.mk:5: "+
			"Please write \"USE_TOOLS+= intltool\" instead of this line.",
		"ERROR: ~/category/package/filename.mk:6: "+
			"\"../../devel/intltool/builtin.mk\" must not be included directly. "+
			"Include \"../../devel/intltool/buildlink3.mk\" instead.")
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
		"ERROR: ~/category/package/hacks.mk:2: " +
			"Relative path \"../../category/package/nonexistent.mk\" does not exist.")
}

func (s *Suite) Test_MkLineChecker_checkInclude__builtin_mk(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		".include \"../../category/package/builtin.mk\"",
		".include \"../../category/package/builtin.mk\" # ok")
	t.CreateFileLines("category/package/builtin.mk",
		MkCvsID)
	t.FinishSetUp()

	G.checkdirPackage(t.File("category/package"))

	t.CheckOutputLines(
		"ERROR: ~/category/package/Makefile:20: " +
			"\"../../category/package/builtin.mk\" must not be included directly. " +
			"Include \"../../category/package/buildlink3.mk\" instead.")
}

func (s *Suite) Test_MkLineChecker_checkInclude__buildlink3_mk_includes_builtin_mk(c *check.C) {
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

func (s *Suite) Test_MkLineChecker_checkInclude__builtin_mk_rationale(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"# I have good reasons for including this file directly.",
		".include \"../../category/package/builtin.mk\"",
		"",
		".include \"../../category/package/builtin.mk\"")
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
	mklines := NewMkLines(lines)

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
		"WARN: ~/category/package/module.mk:5: LATEST_PYTHON is used but not defined.",
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
			"should look like \"../../category/package\", not \"../package\".")
}

func (s *Suite) Test_MkLineChecker_CheckRelativePkgdir(c *check.C) {
	t := s.Init(c)

	t.CreateFileLines("other/package/Makefile")

	test := func(relativePkgdir RelPath, diagnostics ...string) {
		// Must be in the filesystem because of directory references.
		mklines := t.SetUpFileMkLines("category/package/Makefile",
			"# dummy")

		checkRelativePkgdir := func(mkline *MkLine) {
			MkLineChecker{mklines, mkline}.CheckRelativePkgdir(relativePkgdir)
		}

		mklines.ForEach(checkRelativePkgdir)

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
			"Please use \".if defined(FNAME_MK)\" instead.",
		"WARN: category/package/filename.mk:12: The \".ifndef\" directive is deprecated. "+
			"Please use \".if !defined(FNAME_MK)\" instead.",
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

func (s *Suite) Test_MkLineChecker_checkDirectiveEnd__ending_comments(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("opsys.mk",
		MkCvsID,
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
		".endif # NetBSD", // Wrong, should be FreeBSD from the .elif.
		"",
		".for ii in 1 2",
		".  for jj in 1 2",
		".  endfor # ii", // Note: a simple "i" would not generate a warning because it is found in the word "in".
		".endfor # ii")

	// See MkLineChecker.checkDirective
	mklines.Check()

	t.CheckOutputLines(
		"WARN: opsys.mk:7: Comment \"MACHINE_ARCH\" does not match condition \"${OS_VERSION:M8.*}\".",
		"WARN: opsys.mk:8: Comment \"OS_VERSION\" does not match condition \"${MACHINE_ARCH} == x86_64\".",
		"WARN: opsys.mk:10: Comment \"j\" does not match loop \"i in 1 2 3 4 5\".",
		"WARN: opsys.mk:12: Unknown option \"option\".",
		"WARN: opsys.mk:20: Comment \"NetBSD\" does not match condition \"${OPSYS} == FreeBSD\".",
		"WARN: opsys.mk:24: Comment \"ii\" does not match loop \"jj in 1 2\".")
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

func (s *Suite) Test_MkLineChecker_checkDirectiveCond(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	test := func(cond string, output ...string) {
		mklines := t.NewMkLines("filename.mk",
			MkCvsID,
			cond,
			".endif")
		mklines.Check()
		t.CheckOutput(output)
	}

	test(
		".if !empty(PKGSRC_COMPILER:Mmycc)",
		"WARN: filename.mk:2: The pattern \"mycc\" cannot match any of "+
			"{ ccache ccc clang distcc f2c gcc hp icc ido "+
			"mipspro mipspro-ucode pcc sunpro xlc } for PKGSRC_COMPILER.")

	test(
		".if ${A} != ${B}",
		"WARN: filename.mk:2: A is used but not defined.",
		"WARN: filename.mk:2: B is used but not defined.")

	test(".if ${HOMEPAGE} == \"mailto:someone@example.org\"",
		"WARN: filename.mk:2: \"mailto:someone@example.org\" is not a valid URL.",
		"WARN: filename.mk:2: HOMEPAGE should not be used at load time in any file.")

	test(".if !empty(PKGSRC_RUN_TEST:M[Y][eE][sS])",
		"WARN: filename.mk:2: PKGSRC_RUN_TEST should be matched "+
			"against \"[yY][eE][sS]\" or \"[nN][oO]\", not \"[Y][eE][sS]\".")

	test(".if !empty(IS_BUILTIN.Xfixes:M[yY][eE][sS])")

	test(".if !empty(${IS_BUILTIN.Xfixes:M[yY][eE][sS]})",
		"WARN: filename.mk:2: The empty() function takes a variable name as parameter, "+
			"not a variable expression.")

	test(".if ${PKGSRC_COMPILER} == \"msvc\"",
		"WARN: filename.mk:2: \"msvc\" is not valid for PKGSRC_COMPILER. "+
			"Use one of { ccache ccc clang distcc f2c gcc hp icc ido mipspro mipspro-ucode pcc sunpro xlc } instead.",
		"ERROR: filename.mk:2: Use ${PKGSRC_COMPILER:Mmsvc} instead of the == operator.")

	test(".if ${PKG_LIBTOOL:Mlibtool}",
		"NOTE: filename.mk:2: PKG_LIBTOOL "+
			"should be compared using \"${PKG_LIBTOOL} == libtool\" "+
			"instead of matching against \":Mlibtool\".",
		"WARN: filename.mk:2: PKG_LIBTOOL should not be used at load time in any file.")

	test(".if ${MACHINE_PLATFORM:MUnknownOS-*-*} || ${MACHINE_ARCH:Mx86}",
		"WARN: filename.mk:2: "+
			"The pattern \"UnknownOS\" cannot match any of "+
			"{ AIX BSDOS Bitrig Cygwin Darwin DragonFly FreeBSD FreeMiNT GNUkFreeBSD HPUX Haiku "+
			"IRIX Interix Linux Minix MirBSD NetBSD OSF1 OpenBSD QNX SCO_SV SunOS UnixWare "+
			"} for the operating system part of MACHINE_PLATFORM.",
		"WARN: filename.mk:2: "+
			"The pattern \"x86\" cannot match any of "+
			"{ aarch64 aarch64eb alpha amd64 arc arm arm26 arm32 cobalt coldfire convex dreamcast earm "+
			"earmeb earmhf earmhfeb earmv4 earmv4eb earmv5 earmv5eb earmv6 earmv6eb earmv6hf earmv6hfeb "+
			"earmv7 earmv7eb earmv7hf earmv7hfeb evbarm hpcmips hpcsh hppa hppa64 i386 i586 i686 ia64 "+
			"m68000 m68k m88k mips mips64 mips64eb mips64el mipseb mipsel mipsn32 mlrisc ns32k pc532 pmax "+
			"powerpc powerpc64 rs6000 s390 sh3eb sh3el sparc sparc64 vax x86_64 "+
			"} for MACHINE_ARCH.",
		"NOTE: filename.mk:2: MACHINE_ARCH "+
			"should be compared using \"${MACHINE_ARCH} == x86\" "+
			"instead of matching against \":Mx86\".")

	// Doesn't occur in practice since it is surprising that the ! applies
	// to the comparison operator, and not to one of its arguments.
	test(".if !${VAR} == value",
		"WARN: filename.mk:2: VAR is used but not defined.")

	// Doesn't occur in practice since this string can never be empty.
	test(".if !\"${VAR}str\"",
		"WARN: filename.mk:2: VAR is used but not defined.")

	// Doesn't occur in practice since !${VAR} && !${VAR2} is more idiomatic.
	test(".if !\"${VAR}${VAR2}\"",
		"WARN: filename.mk:2: VAR is used but not defined.",
		"WARN: filename.mk:2: VAR2 is used but not defined.")

	// Just for code coverage; always evaluates to true.
	test(".if \"string\"",
		nil...)

	// Code coverage for checkVar.
	test(".if ${OPSYS} || ${MACHINE_ARCH}",
		nil...)

	test(".if ${VAR}",
		"WARN: filename.mk:2: VAR is used but not defined.")

	test(".if ${VAR} == 3",
		"WARN: filename.mk:2: VAR is used but not defined.")

	test(".if \"value\" == ${VAR}",
		"WARN: filename.mk:2: VAR is used but not defined.")

	test(".if ${MASTER_SITES:Mftp://*} == \"ftp://netbsd.org/\"",
		// FIXME: duplicate diagnostic, see MkParser.MkCond.
		"WARN: filename.mk:2: Invalid variable modifier \"//*\" for \"MASTER_SITES\".",
		"WARN: filename.mk:2: Invalid variable modifier \"//*\" for \"MASTER_SITES\".",
		"WARN: filename.mk:2: \"ftp\" is not a valid URL.",
		"WARN: filename.mk:2: MASTER_SITES should not be used at load time in any file.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCond__tracing(c *check.C) {
	t := s.Init(c)

	t.EnableTracingToLog()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".if ${VAR:Mpattern1:Mpattern2} == comparison",
		".endif")

	mklines.Check()

	t.CheckOutputLinesMatching(`^WARN|checkCompare`,
		"TRACE: 1 2   checkCompareVarStr ${VAR:Mpattern1:Mpattern2} == comparison",
		"WARN: filename.mk:2: VAR is used but not defined.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCond__comparison_with_shell_command(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("security/openssl/Makefile",
		MkCvsID,
		".if ${PKGSRC_COMPILER} == \"gcc\" && ${CC} == \"cc\"",
		".endif")

	mklines.Check()

	// Don't warn about unknown shell command "cc".
	t.CheckOutputLines(
		"ERROR: security/openssl/Makefile:2: Use ${PKGSRC_COMPILER:Mgcc} instead of the == operator.")
}

// The :N modifier filters unwanted values. After this filter, any variable value
// may be compared with the empty string, regardless of the variable type.
// Effectively, the :N modifier changes the type from T to Option(T).
func (s *Suite) Test_MkLineChecker_checkDirectiveCond__compare_pattern_with_empty(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		".if ${X11BASE:Npattern} == \"\"",
		".endif",
		"",
		".if ${X11BASE:N<>} == \"*\"",
		".endif",
		"",
		".if !(${OPSYS:M*BSD} != \"\")",
		".endif")

	mklines.Check()

	// TODO: There should be a warning about "<>" containing invalid
	//  characters for a path. See VartypeCheck.Pathname
	t.CheckOutputLines(
		"WARN: filename.mk:5: The pathname pattern \"<>\" contains the invalid characters \"<>\".",
		"WARN: filename.mk:5: The pathname \"*\" contains the invalid character \"*\".")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCond__comparing_PKGSRC_COMPILER_with_eqeq(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		".if ${PKGSRC_COMPILER} == \"clang\"",
		".elif ${PKGSRC_COMPILER} != \"gcc\"",
		".endif")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: Makefile:2: Use ${PKGSRC_COMPILER:Mclang} instead of the == operator.",
		"ERROR: Makefile:3: Use ${PKGSRC_COMPILER:Ngcc} instead of the != operator.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCondEmpty(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	t.Chdir("category/package")
	t.FinishSetUp()

	// before: the directive before the condition is simplified
	// after: the directive after the condition is simplified
	// diagnostics: the usual ones
	test := func(before, after string, diagnostics ...string) {
		mklines := t.SetUpFileMkLines("module.mk",
			MkCvsID,
			"",
			before,
			".endif")

		t.ExpectDiagnosticsAutofix(
			mklines.Check,
			diagnostics...)

		afterMklines := LoadMk(t.File("module.mk"), MustSucceed)
		t.CheckEquals(afterMklines.mklines[2].Text, after)
	}

	test(
		".if !empty(OPSYS:MUnknown)",
		".if ${OPSYS} == Unknown",

		"WARN: module.mk:3: The pattern \"Unknown\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.",
		"NOTE: module.mk:3: OPSYS should be "+
			"compared using \"${OPSYS} == Unknown\" "+
			"instead of matching against \":MUnknown\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(OPSYS:MUnknown)\" "+
			"with \"${OPSYS} == Unknown\".")

	// The condition can only be simplified if the :M or :N modifier is
	// the last one on the chain.
	test(
		".if !empty(OPSYS:O:MUnknown:S,a,b,)",
		".if !empty(OPSYS:O:MUnknown:S,a,b,)",

		"WARN: module.mk:3: The pattern \"Unknown\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCondEmptyType(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	t.Chdir("category/package")
	t.FinishSetUp()

	test := func(line string, diagnostics ...string) {
		mklines := t.SetUpFileMkLines("module.mk",
			MkCvsID,
			"",
			line,
			".endif")

		mklines.ForEach(func(mkline *MkLine) {
			ck := NewMkLineChecker(mklines, mkline)
			mkline.ForEachUsed(func(varUse *MkVarUse, time VucTime) {
				ck.checkDirectiveCondEmptyType(varUse)
			})
		})

		t.CheckOutput(diagnostics)
	}

	test(".if !empty(OPSYS:Mok)",
		"WARN: module.mk:3: The pattern \"ok\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.")

	// As of December 2019, pkglint doesn't analyze the :S modifier in
	// depth and therefore simply skips the type check for the :M
	// modifier.
	test(".if !empty(OPSYS:S,NetBSD,ok,:Mok)",
		nil...)
	test(".if !empty(OPSYS:C,NetBSD,ok,:Mok)",
		nil...)

	// Several other modifiers are ok since they don't modify the
	// individual words.
	test(".if !empty(OPSYS:O:u:Mok)",
		"WARN: module.mk:3: The pattern \"ok\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.")

	// Other modifiers can modify the words themselves. As long as
	// pkglint doesn't actually evaluate these modifiers, suppress
	// any warnings.
	test(".if !empty(OPSYS:E:Mok)",
		nil...)
	test(".if !empty(OPSYS:H:Mok)",
		nil...)
	test(".if !empty(OPSYS:R:Mok)",
		nil...)
	test(".if !empty(OPSYS:tl:Mok)",
		nil...)
	test(".if !empty(OPSYS:tW:Mok)",
		nil...)
	test(".if !empty(OPSYS:tW:Mok)",
		nil...)
}

func (s *Suite) Test_MkLineChecker_simplifyCondition(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	t.Chdir("category/package")
	t.FinishSetUp()

	// before: the directive before the condition is simplified
	// after: the directive after the condition is simplified
	// diagnostics: the usual ones
	test := func(before, after string, diagnostics ...string) {
		mklines := t.SetUpFileMkLines("module.mk",
			MkCvsID,
			"",
			before,
			".endif")

		t.ExpectDiagnosticsAutofix(
			mklines.Check,
			diagnostics...)

		afterMklines := LoadMk(t.File("module.mk"), MustSucceed)
		t.CheckEquals(afterMklines.mklines[2].Text, after)
	}

	test(
		".if ${PKGPATH:Mpattern}",
		".if ${PKGPATH} == pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mpattern}\" "+
			"with \"${PKGPATH} == pattern\".")

	// When the pattern contains placeholders, it cannot be converted to == or !=.
	test(
		".if ${PKGPATH:Mpa*n}",
		".if ${PKGPATH:Mpa*n}",

		nil...)

	// When deciding whether to replace the expression, only the
	// last modifier is inspected. All the others are copied.
	test(
		".if ${PKGPATH:tl:Mpattern}",
		".if ${PKGPATH:tl} == pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH:tl} == pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:tl:Mpattern}\" "+
			"with \"${PKGPATH:tl} == pattern\".")

	// Negated pattern matches are supported as well,
	// as long as the variable is guaranteed to be nonempty.
	test(
		".if ${PKGPATH:Ncategory/package}",
		".if ${PKGPATH} != category/package",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != category/package\" "+
			"instead of matching against \":Ncategory/package\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Ncategory/package}\" "+
			"with \"${PKGPATH} != category/package\".")

	// ${PKGPATH:None:Ntwo} is a short variant of ${PKGPATH} != "one" &&
	// ${PKGPATH} != "two". Applying the transformation would make the
	// condition longer than before, therefore nothing is done here.
	test(
		".if ${PKGPATH:None:Ntwo}",
		".if ${PKGPATH:None:Ntwo}",

		nil...)

	// Note: this combination doesn't make sense since the patterns
	// "one" and "two" don't overlap.
	test(
		".if ${PKGPATH:Mone:Mtwo}",
		".if ${PKGPATH:Mone} == two",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH:Mone} == two\" "+
			"instead of matching against \":Mtwo\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mone:Mtwo}\" "+
			"with \"${PKGPATH:Mone} == two\".")

	test(
		".if !empty(PKGPATH:Mpattern)",
		".if ${PKGPATH} == pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(PKGPATH:Mpattern)\" "+
			"with \"${PKGPATH} == pattern\".")

	test(
		".if empty(PKGPATH:Mpattern)",
		".if ${PKGPATH} != pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"empty(PKGPATH:Mpattern)\" "+
			"with \"${PKGPATH} != pattern\".")

	test(
		".if !!empty(PKGPATH:Mpattern)",
		// TODO: The ! and == could be combined into a !=.
		//  Luckily the !! pattern doesn't occur in practice.
		".if !${PKGPATH} == pattern",

		// TODO: When taking all the ! into account, this is actually a
		//  test for emptiness, therefore the diagnostics should suggest
		//  the != operator instead of ==.
		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(PKGPATH:Mpattern)\" "+
			"with \"${PKGPATH} == pattern\".")

	test(".if empty(PKGPATH:Mpattern) || 0",
		".if ${PKGPATH} != pattern || 0",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"empty(PKGPATH:Mpattern)\" "+
			"with \"${PKGPATH} != pattern\".")

	// No note in this case since there is no implicit !empty around the varUse.
	test(
		".if ${PKGPATH:Mpattern} != ${OTHER}",
		".if ${PKGPATH:Mpattern} != ${OTHER}",

		"WARN: module.mk:3: OTHER is used but not defined.")

	test(
		".if ${PKGPATH:Mpattern}",
		".if ${PKGPATH} == pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mpattern}\" "+
			"with \"${PKGPATH} == pattern\".")

	test(
		".if !${PKGPATH:Mpattern}",
		".if ${PKGPATH} != pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"!${PKGPATH:Mpattern}\" "+
			"with \"${PKGPATH} != pattern\".")

	// TODO: Merge the double negation into the comparison operator.
	test(
		".if !!${PKGPATH:Mpattern}",
		".if !${PKGPATH} != pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != pattern\" "+
			"instead of matching against \":Mpattern\".",
		"AUTOFIX: module.mk:3: Replacing \"!${PKGPATH:Mpattern}\" "+
			"with \"${PKGPATH} != pattern\".")

	// This pattern with spaces doesn't make sense at all in the :M
	// modifier since it can never match.
	// Or can it, if the PKGPATH contains quotes?
	// How exactly does bmake apply the matching here, are both values unquoted?
	test(
		".if ${PKGPATH:Mpattern with spaces}",
		".if ${PKGPATH:Mpattern with spaces}",

		"WARN: module.mk:3: The pathname pattern \"pattern with spaces\" "+
			"contains the invalid characters \"  \".")
	// TODO: ".if ${PKGPATH} == \"pattern with spaces\"")

	test(
		".if ${PKGPATH:M'pattern with spaces'}",
		".if ${PKGPATH:M'pattern with spaces'}",

		"WARN: module.mk:3: The pathname pattern \"'pattern with spaces'\" "+
			"contains the invalid characters \"'  '\".")
	// TODO: ".if ${PKGPATH} == 'pattern with spaces'")

	test(
		".if ${PKGPATH:M&&}",
		".if ${PKGPATH:M&&}",

		"WARN: module.mk:3: The pathname pattern \"&&\" "+
			"contains the invalid characters \"&&\".")
	// TODO: ".if ${PKGPATH} == '&&'")

	// If PKGPATH is "", the condition is false.
	// If PKGPATH is "negative-pattern", the condition is false.
	// In all other cases, the condition is true.
	//
	// Therefore this condition cannot simply be transformed into
	// ${PKGPATH} != negative-pattern, since that would produce a
	// different result in the case where PKGPATH is empty.
	//
	// For system-provided variables that are guaranteed to be non-empty,
	// such as OPSYS or PKGPATH, this replacement is valid.
	// These variables are only guaranteed to be defined after bsd.prefs.mk
	// has been included, like everywhere else.
	test(
		".if ${PKGPATH:Nnegative-pattern}",
		".if ${PKGPATH} != negative-pattern",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} != negative-pattern\" "+
			"instead of matching against \":Nnegative-pattern\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Nnegative-pattern}\" "+
			"with \"${PKGPATH} != negative-pattern\".")

	// Since UNKNOWN is not a well-known system-provided variable that is
	// guaranteed to be non-empty (see the previous example), it is not
	// transformed at all.
	test(
		".if ${UNKNOWN:Nnegative-pattern}",
		".if ${UNKNOWN:Nnegative-pattern}",

		"WARN: module.mk:3: UNKNOWN is used but not defined.")

	test(
		".if ${PKGPATH:Mpath1} || ${PKGPATH:Mpath2}",
		".if ${PKGPATH} == path1 || ${PKGPATH} == path2",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == path1\" "+
			"instead of matching against \":Mpath1\".",
		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == path2\" "+
			"instead of matching against \":Mpath2\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mpath1}\" "+
			"with \"${PKGPATH} == path1\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mpath2}\" "+
			"with \"${PKGPATH} == path2\".")

	test(
		".if (((((${PKGPATH:Mpath})))))",
		".if (((((${PKGPATH} == path)))))",

		"NOTE: module.mk:3: PKGPATH "+
			"should be compared using \"${PKGPATH} == path\" "+
			"instead of matching against \":Mpath\".",
		"AUTOFIX: module.mk:3: Replacing \"${PKGPATH:Mpath}\" "+
			"with \"${PKGPATH} == path\".")

	test(
		".if ${MACHINE_ARCH:Mx86_64}",
		".if ${MACHINE_ARCH} == x86_64",

		"NOTE: module.mk:3: MACHINE_ARCH "+
			"should be compared using \"${MACHINE_ARCH} == x86_64\" "+
			"instead of matching against \":Mx86_64\".",
		"AUTOFIX: module.mk:3: Replacing \"${MACHINE_ARCH:Mx86_64}\" "+
			"with \"${MACHINE_ARCH} == x86_64\".")

	test(
		".if !empty(OPSYS:MUnknown)",
		".if ${OPSYS} == Unknown",

		// FIXME: This warning is not the job of simplifyCondition.
		"WARN: module.mk:3: The pattern \"Unknown\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.",
		"NOTE: module.mk:3: OPSYS should be "+
			"compared using \"${OPSYS} == Unknown\" "+
			"instead of matching against \":MUnknown\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(OPSYS:MUnknown)\" "+
			"with \"${OPSYS} == Unknown\".")

	test(
		".if !empty(OPSYS:S,NetBSD,ok,:Mok)",
		".if ${OPSYS:S,NetBSD,ok,} == ok",

		"NOTE: module.mk:3: OPSYS should be "+
			"compared using \"${OPSYS:S,NetBSD,ok,} == ok\" "+
			"instead of matching against \":Mok\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(OPSYS:S,NetBSD,ok,:Mok)\" "+
			"with \"${OPSYS:S,NetBSD,ok,} == ok\".")

	test(
		".if empty(OPSYS:tl:Msunos)",
		".if ${OPSYS:tl} != sunos",

		"NOTE: module.mk:3: OPSYS should be "+
			"compared using \"${OPSYS:tl} != sunos\" "+
			"instead of matching against \":Msunos\".",
		"AUTOFIX: module.mk:3: Replacing \"empty(OPSYS:tl:Msunos)\" "+
			"with \"${OPSYS:tl} != sunos\".")

	test(
		".if !empty(OPSYS:O:MUnknown:S,a,b,)",
		".if !empty(OPSYS:O:MUnknown:S,a,b,)",

		"WARN: module.mk:3: The pattern \"Unknown\" cannot match any of "+
			"{ Cygwin DragonFly FreeBSD Linux NetBSD SunOS } for OPSYS.")

	// TODO: Allow the dot without quotes.
	test(
		".if !empty(PKGPATH:Mcategory/package1.2)",
		".if ${PKGPATH} == category/package1.2",

		"NOTE: module.mk:3: PKGPATH should be "+
			"compared using \"${PKGPATH} == category/package1.2\" "+
			"instead of matching against \":Mcategory/package1.2\".",
		"AUTOFIX: module.mk:3: Replacing \"!empty(PKGPATH:Mcategory/package1.2)\" "+
			"with \"${PKGPATH} == category/package1.2\".")

	// Numbers must be enclosed in quotes, otherwise they are compared
	// as numbers, not as strings. The :M and :N modifiers always compare
	// strings.
	test(
		".if empty(ABI:U:M64)",
		".if ${ABI:U} != \"64\"",

		"NOTE: module.mk:3: ABI should be compared using \"${ABI:U} != \"64\"\" "+
			"instead of matching against \":M64\".",
		"AUTOFIX: module.mk:3: Replacing \"empty(ABI:U:M64)\" "+
			"with \"${ABI:U} != \\\"64\\\"\".")

	// Fractional numbers must also be enclosed in quotes.
	test(
		".if empty(PKGVERSION_NOREV:U:M19.12)",
		".if ${PKGVERSION_NOREV:U} != \"19.12\"",

		"NOTE: module.mk:3: PKGVERSION_NOREV should be "+
			"compared using \"${PKGVERSION_NOREV:U} != \"19.12\"\" "+
			"instead of matching against \":M19.12\".",
		"WARN: module.mk:3: PKGVERSION_NOREV should not be used at load time in any file.",
		"AUTOFIX: module.mk:3: Replacing \"empty(PKGVERSION_NOREV:U:M19.12)\" "+
			"with \"${PKGVERSION_NOREV:U} != \\\"19.12\\\"\".")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCondCompare(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	test := func(cond string, output ...string) {
		mklines := t.NewMkLines("filename.mk",
			cond)
		mklines.ForEach(func(mkline *MkLine) {
			MkLineChecker{mklines, mkline}.checkDirectiveCond()
		})
		t.CheckOutput(output)
	}

	// As of July 2019, pkglint doesn't have specific checks for comparing
	// variables to numbers.
	test(".if ${VAR} > 0",
		"WARN: filename.mk:1: VAR is used but not defined.")

	// For string comparisons, the checks from vartypecheck.go are
	// performed.
	test(".if ${DISTNAME} == \"<>\"",
		"WARN: filename.mk:1: The filename \"<>\" contains the invalid characters \"<>\".",
		"WARN: filename.mk:1: DISTNAME should not be used at load time in any file.")

	// This type of comparison doesn't occur in practice since it is
	// overly verbose.
	test(".if \"${BUILD_DIRS}str\" == \"str\"",
		// TODO: why should it not be used? In a .for loop it sounds pretty normal.
		"WARN: filename.mk:1: BUILD_DIRS should not be used at load time in any file.")

	// This is a shorthand for defined(VAR), but it is not used in practice.
	test(".if VAR",
		"WARN: filename.mk:1: Invalid condition, unrecognized part: \"VAR\".")

	// Calling a function with braces instead of parentheses is syntactically
	// invalid. Pkglint is stricter than bmake in this situation.
	//
	// Bmake reads the "empty{VAR}" as a variable name. It then checks whether
	// this variable is defined. It is not, of course, therefore the expression
	// is false. The ! in front of it negates this false, which makes the whole
	// condition true.
	//
	// See https://mail-index.netbsd.org/tech-pkg/2019/07/07/msg021539.html
	test(".if !empty{VAR}",
		"WARN: filename.mk:1: Invalid condition, unrecognized part: \"empty{VAR}\".")
}

func (s *Suite) Test_MkLineChecker_checkDirectiveCondCompareVarStr__no_tracing(c *check.C) {
	t := s.Init(c)
	b := NewMkTokenBuilder()

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		".if ${DISTFILES:Mpattern:O:u} == NetBSD")
	t.DisableTracing()

	ck := MkLineChecker{mklines, mklines.mklines[0]}
	varUse := b.VarUse("DISTFILES", "Mpattern", "O", "u")
	ck.checkDirectiveCondCompareVarStr(varUse, "==", "distfile-1.0.tar.gz")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_MkLineChecker_checkCompareVarStrCompiler(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.Chdir(".")

	test := func(cond string, diagnostics ...string) {
		mklines := t.SetUpFileMkLines("filename.mk",
			MkCvsID,
			"",
			".if "+cond,
			".endif")

		t.SetUpCommandLine("-Wall")
		mklines.Check()
		t.SetUpCommandLine("-Wall", "--autofix")
		mklines.Check()

		t.CheckOutput(diagnostics)
	}

	test(
		"${PKGSRC_COMPILER} == gcc",

		"ERROR: filename.mk:3: "+
			"Use ${PKGSRC_COMPILER:Mgcc} instead of the == operator.",
		"AUTOFIX: filename.mk:3: "+
			"Replacing \"${PKGSRC_COMPILER} == gcc\" "+
			"with \"${PKGSRC_COMPILER:Mgcc}\".")

	// No autofix because of missing whitespace.
	// TODO: Provide the autofix regardless of the whitespace.
	test(
		"${PKGSRC_COMPILER}==gcc",

		"ERROR: filename.mk:3: "+
			"Use ${PKGSRC_COMPILER:Mgcc} instead of the == operator.")

	// The comparison value can be with or without quotes.
	test(
		"${PKGSRC_COMPILER} == \"gcc\"",

		"ERROR: filename.mk:3: "+
			"Use ${PKGSRC_COMPILER:Mgcc} instead of the == operator.",
		"AUTOFIX: filename.mk:3: "+
			"Replacing \"${PKGSRC_COMPILER} == \\\"gcc\\\"\" "+
			"with \"${PKGSRC_COMPILER:Mgcc}\".")

	// No warning because it is not obvious what is meant here.
	// This case probably doesn't occur in practice.
	test(
		"${PKGSRC_COMPILER} == \"distcc gcc\"",

		nil...)
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
		".endfor")

	mklines.Check()

	t.CheckOutputLines(
		// No warning about a missing :Q in line 2 since the :C modifier
		// converts the colon-separated list into a space-separated list,
		// as required by the .for loop.

		// This warning is correct since PATH is separated by colons, not by spaces.
		"WARN: for.mk:5: Please use ${PATH:Q} instead of ${PATH}.",

		// This warning is also correct since the :M modifier doesn't
		// turn a list into a non-list or vice versa.
		"WARN: for.mk:8: Please use ${PATH:M*/bin:Q} instead of ${PATH:M*/bin}.")
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
