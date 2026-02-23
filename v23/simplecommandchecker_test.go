package pkglint

import "gopkg.in/check.v1"

func (s *Suite) Test_SimpleCommandChecker__case_continue_with_loop(c *check.C) {
	t := s.Init(c)

	code := "case $$fname in ${CHECK_PORTABILITY_SKIP:@p@${p}) continue;; @} esac"
	line := t.NewLine("filename.mk", 123, "\t"+code)

	program, err := parseShellProgram(line, code)
	assertNil(err, "parse error")
	t.CheckEquals(
		program.AndOrs[0].Pipes[0].Cmds[0].Compound.Case.Cases[0].Var.MkText,
		"${CHECK_PORTABILITY_SKIP:@p@${p}) continue;; @}")
}

func (s *Suite) Test_SimpleCommandChecker__case_continue_with_suffix(c *check.C) {
	t := s.Init(c)

	code := "case $$fname in ${CHECK_PORTABILITY_SKIP:=) continue;; } esac"
	line := t.NewLine("filename.mk", 123, "\t"+code)

	program, err := parseShellProgram(line, code)
	assertNil(err, "parse error: parse error at []string{\"esac\"}")

	t.CheckEquals(
		program.AndOrs[0].Pipes[0].Cmds[0].Compound.Case.Cases[0].Var.MkText,
		"${CHECK_PORTABILITY_SKIP:=) continue;; }")
}

// When pkglint is called without -Wextra, the check for unknown shell
// commands is disabled, as it is still unreliable. As of December 2019
// there are around 500 warnings in pkgsrc, and several of them are wrong.
func (s *Suite) Test_SimpleCommandChecker_checkCommandStart__unknown_default(c *check.C) {
	t := s.Init(c)

	var pkg *Package
	test := func(commandLineArg string, diagnostics ...string) {
		t.SetUpCommandLine(commandLineArg)
		mklines := t.NewMkLinesPkg("Makefile", pkg,
			MkCvsID,
			"",
			"MY_TOOL.i386=\t${PREFIX}/bin/tool-i386",
			"MY_TOOL.x86_64=\t${PREFIX}/bin/tool-x86_64",
			"",
			"pre-configure:",
			"\t${MY_TOOL.amd64} -e 'print 12345'",
			"\t${UNKNOWN_TOOL}")

		mklines.Check()

		t.CheckOutput(diagnostics)
	}

	t.SetUpPackage("category/package")
	pkg = NewPackage(t.File("category/package"))
	t.Chdir("category/package")
	t.FinishSetUp()

	test(".", // Override the default -Wall option.
		nil...)

	test("-Wall,no-extra",
		nil...)

	test("-Wall",
		"WARN: Makefile:8: Unknown shell command \"${UNKNOWN_TOOL}\".",
		"WARN: Makefile:8: Variable \"UNKNOWN_TOOL\" is used but not defined.")
}

// Despite its name, the TOOLS_PATH.* name the whole shell command,
// not just the path of its executable.
func (s *Suite) Test_SimpleCommandChecker_checkCommandStart__TOOLS_PATH(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"CONFIG_SHELL=\t${TOOLS_PATH.bash}")
	t.Chdir("category/package")
	t.FinishSetUp()
	G.checkdirPackage(".")

	t.CheckOutputEmpty()
}

func (s *Suite) Test_SimpleCommandChecker_checkInstallCommand(c *check.C) {
	t := s.Init(c)

	lines := func(lines ...string) []string { return lines }

	test := func(lines []string, diagnostics ...string) {
		mklines := t.NewMkLines("filename.mk",
			mapStr(lines, func(s string) string { return "\t" + s })...)
		mklines.checkAllData.target = "do-install"

		mklines.ForEach(func(mkline *MkLine) {
			program, err := parseShellProgram(nil, mkline.ShellCommand())
			assertNil(err, "")

			walker := NewMkShWalker()
			walker.Callback.SimpleCommand = func(command *MkShSimpleCommand) {
				scc := NewSimpleCommandChecker(command, RunTime, mkline, mklines)
				scc.checkInstallCommand(command.Name.MkText)
			}
			walker.Walk(program)
		})

		t.CheckOutput(diagnostics)
	}

	test(
		lines(
			"sed",
			"${SED}"),
		"WARN: filename.mk:1: The shell command \"sed\" "+
			"should not be used in the install phase.",
		"WARN: filename.mk:2: The shell command \"${SED}\" "+
			"should not be used in the install phase.")

	test(
		lines(
			"tr",
			"${TR}"),
		"WARN: filename.mk:1: The shell command \"tr\" "+
			"should not be used in the install phase.",
		"WARN: filename.mk:2: The shell command \"${TR}\" "+
			"should not be used in the install phase.")

	test(
		lines(
			"cp",
			"${CP}"),
		"WARN: filename.mk:1: ${CP} should not be used to install files.",
		"WARN: filename.mk:2: ${CP} should not be used to install files.")

	test(
		lines(
			"${INSTALL}",
			"${INSTALL_DATA}",
			"${INSTALL_DATA_DIR}",
			"${INSTALL_LIB}",
			"${INSTALL_LIB_DIR}",
			"${INSTALL_MAN}",
			"${INSTALL_MAN_DIR}",
			"${INSTALL_PROGRAM}",
			"${INSTALL_PROGRAM_DIR}",
			"${INSTALL_SCRIPT}",
			"${LIBTOOL}",
			"${LN}",
			"${PAX}"),
		nil...)
}

func (s *Suite) Test_SimpleCommandChecker_handleForbiddenCommand(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"\t${RUN} mktexlsr; texconfig")

	mklines.Check()

	t.CheckOutputLines(
		"ERROR: Makefile:3: \"mktexlsr\" must not be used in Makefiles.",
		"ERROR: Makefile:3: \"texconfig\" must not be used in Makefiles.")
}

func (s *Suite) Test_SimpleCommandChecker_handleCommandVariable(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("runtime", "RUNTIME", AtRunTime)
	t.SetUpTool("nowhere", "NOWHERE", Nowhere)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"RUNTIME_Q_CMD=\t${RUNTIME:Q}",
		"NOWHERE_Q_CMD=\t${NOWHERE:Q}",
		"RUNTIME_CMD=\t${RUNTIME}",
		"NOWHERE_CMD=\t${NOWHERE}",
		"",
		"pre-configure:",
		"\t: ${RUNTIME_Q_CMD} ${NOWHERE_Q_CMD}",
		"\t: ${RUNTIME_CMD} ${NOWHERE_CMD}",
		"\t${PKGNAME}") // This doesn't make sense; it's just for code coverage

	mklines.Check()

	// A tool that appears as the name of a shell command is exactly
	// intended to be used without quotes, so that its possible
	// command line options are treated as separate arguments.
	//
	// TODO: Add a warning that in lines 3 and 4, the :Q is wrong.
	t.CheckOutputLines(
		"WARN: Makefile:4: The \"${NOWHERE:Q}\" tool is used but not added to USE_TOOLS.",
		"WARN: Makefile:6: The \"${NOWHERE}\" tool is used but not added to USE_TOOLS.",
		"WARN: Makefile:11: Unknown shell command \"${PKGNAME}\".")
}

func (s *Suite) Test_SimpleCommandChecker_handleCommandVariable__parameterized(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	pkg := NewPackage(t.File("category/package"))
	t.FinishSetUp()

	mklines := t.NewMkLinesPkg("Makefile", pkg,
		MkCvsID,
		"",
		"MY_TOOL.i386=\t${PREFIX}/bin/tool-i386",
		"MY_TOOL.x86_64=\t${PREFIX}/bin/tool-x86_64",
		"",
		"pre-configure:",
		"\t${MY_TOOL.amd64} -e 'print 12345'",
		"\t${UNKNOWN_TOOL}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:8: Unknown shell command \"${UNKNOWN_TOOL}\".",
		"WARN: Makefile:8: Variable \"UNKNOWN_TOOL\" is used but not defined.")
}

func (s *Suite) Test_SimpleCommandChecker_handleCommandVariable__followed_by_literal(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package")
	pkg := NewPackage(t.File("category/package"))
	t.FinishSetUp()

	mklines := t.NewMkLinesPkg("Makefile", pkg,
		MkCvsID,
		"",
		"QTDIR=\t${PREFIX}",
		"",
		"pre-configure:",
		"\t${QTDIR}/bin/release")

	mklines.Check()

	t.CheckOutputEmpty()
}

// The package Makefile and other .mk files in a package directory
// may use any shell commands defined by any included files.
// But only if the package is checked as a whole.
//
// On the contrary, when pkglint checks a single .mk file, these
// commands are not guaranteed to be defined, not even when the
// .mk file includes the file defining the command.
// TODO: This paragraph sounds wrong. All commands from included files should be valid.
//
// The PYTHON_BIN variable below must not be called *_CMD, or another code path is taken.
func (s *Suite) Test_SimpleCommandChecker_handleCommandVariable__from_package(c *check.C) {
	t := s.Init(c)

	pkg := t.SetUpPackage("category/package",
		"post-install:",
		"\t${PYTHON_BIN}",
		"",
		".include \"extra.mk\"")
	t.CreateFileLines("category/package/extra.mk",
		MkCvsID,
		"PYTHON_BIN=\tmy_cmd")
	t.FinishSetUp()

	G.Check(pkg)

	t.CheckOutputEmpty()
}

func (s *Suite) Test_SimpleCommandChecker_handleShellBuiltin(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"\t:")
	mkline := mklines.mklines[0]

	test := func(command string, isBuiltin bool) {
		token := NewShToken(command, NewShAtom(shtText, command, shqPlain))
		simpleCommand := &MkShSimpleCommand{Name: token}
		scc := NewSimpleCommandChecker(simpleCommand, RunTime, mkline, mklines)
		t.CheckEquals(scc.handleShellBuiltin(), isBuiltin)
	}

	test(":", true)
	test("break", true)
	test("cd", true)
	test("continue", true)
	test("eval", true)
	test("exec", true)
	test("exit", true)
	test("export", true)
	test("read", true)
	test("set", true)
	test("shift", true)
	test("umask", true)
	test("unset", true)

	test("git", false)
}

func (s *Suite) Test_SimpleCommandChecker_checkRegexReplace(c *check.C) {
	t := s.Init(c)

	test := func(cmd string, diagnostics ...string) {
		t.SetUpTool("pax", "PAX", AtRunTime)
		t.SetUpTool("sed", "SED", AtRunTime)
		mklines := t.NewMkLines("Makefile",
			MkCvsID,
			"pre-configure:",
			"\t"+cmd)

		mklines.Check()

		t.CheckOutput(diagnostics)
	}

	test("${PAX} -s s,.*,, src dst",
		"WARN: Makefile:3: Substitution commands like \"s,.*,,\" should always be quoted.")

	test("pax -s s,.*,, src dst",
		"WARN: Makefile:3: Substitution commands like \"s,.*,,\" should always be quoted.")

	test("${SED} -e s,.*,, src dst",
		"WARN: Makefile:3: Substitution commands like \"s,.*,,\" should always be quoted.")

	test("sed -e s,.*,, src dst",
		"WARN: Makefile:3: Substitution commands like \"s,.*,,\" should always be quoted.")

	// The * is properly enclosed in quotes.
	test("sed -e 's,.*,,' -e \"s,-*,,\"",
		nil...)

	// The * is properly escaped.
	test("sed -e s,.\\*,,",
		nil...)

	test("pax -s s,\\.orig,, src dst",
		nil...)

	test("sed -e s,a,b,g src dst",
		nil...)

	// TODO: Merge the code with BtSedCommands.

	// TODO: Finally, remove the G.Testing from the main code.
	//  Then, remove this test case.
	G.Testing = false
	test("sed -e s,.*,match,",
		nil...)
	G.Testing = true
}

func (s *Suite) Test_SimpleCommandChecker_checkAutoMkdirs(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	// TODO: Check whether these tools are actually necessary for this test.
	t.SetUpTool("awk", "AWK", AtRunTime)
	t.SetUpTool("cp", "CP", AtRunTime)
	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("mkdir", "MKDIR", AtRunTime) // This is actually "mkdir -p".
	t.SetUpTool("unzip", "UNZIP_CMD", AtRunTime)

	var pkg *Package

	test := func(shellCommand string, diagnostics ...string) {
		mklines := t.NewMkLinesPkg("filename.mk", pkg,
			"\t"+shellCommand)
		ck := NewShellLineChecker(mklines, mklines.mklines[0])

		mklines.ForEach(func(mkline *MkLine) {
			ck.CheckShellCommandLine(ck.mkline.ShellCommand())
		})

		t.CheckOutput(diagnostics)
	}

	// AUTO_MKDIRS applies only when installing directories.
	test("${RUN} ${INSTALL} -c ${WRKSRC}/file ${PREFIX}/bin/",
		nil...)

	// TODO: Warn that ${INSTALL} -d can only handle a single directory.
	test("${RUN} ${INSTALL} -m 0755 -d ${PREFIX}/first ${PREFIX}/second",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= first\" instead of \"${INSTALL} -d\".",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= second\" instead of \"${INSTALL} -d\".")

	pkg = NewPackage(t.File("category/pkgbase"))
	pkg.Plist.UnconditionalDirs["share/pkgbase"] = &PlistLine{
		t.NewLine("PLIST", 123, "share/pkgbase/file"),
		nil,
		"share/pkgbase/file"}

	// A directory that is found in the PLIST.
	// TODO: Add a test for using this command inside a conditional;
	//  the note should not appear then.
	test("${RUN} ${INSTALL_DATA_DIR} share/pkgbase ${PREFIX}/share/pkgbase",
		"NOTE: filename.mk:1: You can use AUTO_MKDIRS=yes or \"INSTALLATION_DIRS+= share/pkgbase\" "+
			"instead of \"${INSTALL_DATA_DIR}\".",
		"WARN: filename.mk:1: The INSTALL_*_DIR commands can only handle one directory at a time.")

	// Directories from .for loops are too dynamic to be replaced with AUTO_MKDIRS.
	// TODO: Expand simple .for loops.
	test("${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/${dir}",
		"WARN: filename.mk:1: Variable \"dir\" is used but not defined.")

	// A directory that is not found in the PLIST would not be created by AUTO_MKDIRS,
	// therefore only INSTALLATION_DIRS is suggested.
	test("${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/share/other",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= share/other\" instead of \"${INSTALL_DATA_DIR}\".")
}

func (s *Suite) Test_SimpleCommandChecker_checkAutoMkdirs__redundant(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"AUTO_MKDIRS=\t\tyes",
		"INSTALLATION_DIRS+=\tshare/redundant",
		"INSTALLATION_DIRS+=\tnot/redundant ${EGDIR}")
	t.CreateFileLines("category/package/PLIST",
		PlistCvsID,
		"share/redundant/file",
		"${EGDIR}/file")

	t.Main("-Wall", "-q", "category/package")

	t.CheckOutputLines(
		"NOTE: ~/category/package/Makefile:21: The directory \"share/redundant\" "+
			"is redundant in INSTALLATION_DIRS.",
		// The below is not proven to be always correct. It assumes that a
		// variable in the Makefile has the same value as the corresponding
		// variable from PLIST_SUBST. Violating this assumption would be
		// confusing to the pkgsrc developers, therefore it's a safe bet.
		// A notable counterexample is PKGNAME in PLIST, which corresponds
		// to PKGNAME_NOREV in the package Makefile.
		"NOTE: ~/category/package/Makefile:22: The directory \"${EGDIR}\" "+
			"is redundant in INSTALLATION_DIRS.")
}

// The AUTO_MKDIRS code in mk/install/install.mk (install-dirs-from-PLIST)
// skips conditional directories, as well as directories with placeholders.
func (s *Suite) Test_SimpleCommandChecker_checkAutoMkdirs__conditional_PLIST(c *check.C) {
	t := s.Init(c)

	t.SetUpPackage("category/package",
		"LIB_SUBDIR=\tsubdir",
		"",
		"do-install:",
		"\t${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/libexec/always",
		"\t${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/libexec/conditional",
		"\t${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/${LIB_SUBDIR}",
	)
	t.Chdir("category/package")
	t.CreateFileLines("PLIST",
		PlistCvsID,
		"libexec/always/always",
		"${LIB_SUBDIR}/file",
		"${PLIST.cond}libexec/conditional/conditional")
	t.FinishSetUp()

	G.checkdirPackage(".")

	// As libexec/conditional will not be created automatically,
	// AUTO_MKDIRS must not be suggested in that line.
	t.CheckOutputLines(
		"NOTE: Makefile:23: You can use AUTO_MKDIRS=yes "+
			"or \"INSTALLATION_DIRS+= libexec/always\" "+
			"instead of \"${INSTALL_DATA_DIR}\".",
		"NOTE: Makefile:24: You can use "+
			"\"INSTALLATION_DIRS+= libexec/conditional\" "+
			"instead of \"${INSTALL_DATA_DIR}\".",
		"NOTE: Makefile:25: You can use "+
			"\"INSTALLATION_DIRS+= ${LIB_SUBDIR}\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")
}

func (s *Suite) Test_SimpleCommandChecker_checkAutoMkdirs__strange_paths(c *check.C) {
	t := s.Init(c)

	test := func(path string, diagnostics ...string) {
		mklines := t.NewMkLines("filename.mk",
			"\t${INSTALL_DATA_DIR} "+path)
		mklines.ForEach(func(mkline *MkLine) {
			program, err := parseShellProgram(nil, mkline.ShellCommand())
			assertNil(err, "")

			walker := NewMkShWalker()
			walker.Callback.SimpleCommand = func(command *MkShSimpleCommand) {
				scc := NewSimpleCommandChecker(command, RunTime, mkline, mklines)
				scc.checkAutoMkdirs()
			}
			walker.Walk(program)
		})
		t.CheckOutput(diagnostics)
	}

	t.Chdir("category/package")

	test("${PREFIX}",
		nil...)

	test("${PREFIX}/",
		nil...)

	test("${PREFIX}//",
		nil...)

	test("${PREFIX}/.",
		nil...)

	test("${PREFIX}//non-canonical",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= non-canonical\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	test("${PREFIX}/non-canonical/////",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= non-canonical\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	test("${PREFIX}/${VAR}",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= ${VAR}\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	test("${PREFIX}/${VAR.param}",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= ${VAR.param}\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	test("${PREFIX}/${.CURDIR}",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= ${.CURDIR}\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	// Internal variables are ok.
	test("${PREFIX}/${_INTERNAL}",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= ${_INTERNAL}\" "+
			"instead of \"${INSTALL_DATA_DIR}\".")

	// Ignore variables from a :@ modifier.
	test("${PREFIX}/${.f.}",
		nil...)

	// Ignore variables from a .for loop.
	test("${PREFIX}/${f}",
		nil...)

	// Ignore variables from a .for loop.
	test("${PREFIX}/${_f_}",
		nil...)

	// Ignore paths containing shell variables as it is hard to
	// predict their values using static analysis.
	test("${PREFIX}/$$f",
		nil...)
}

// This test ensures that the command line options to INSTALL_*_DIR are properly
// parsed and do not lead to "can only handle one directory at a time" warnings.
func (s *Suite) Test_SimpleCommandChecker_checkInstallMulti(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("install.mk",
		MkCvsID,
		"",
		"do-install:",
		"\t${INSTALL_PROGRAM_DIR} -m 0555 -g ${APACHE_GROUP} -o ${APACHE_USER} \\",
		"\t\t${DESTDIR}${PREFIX}/lib/apache-modules")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: install.mk:4--5: You can use \"INSTALLATION_DIRS+= lib/apache-modules\" " +
			"instead of \"${INSTALL_PROGRAM_DIR}\".")
}

func (s *Suite) Test_SimpleCommandChecker_checkPaxPe(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("pax", "PAX", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"do-install:",
		"\t${RUN} pax -pe ${WRKSRC} ${DESTDIR}${PREFIX}",
		"\t${RUN} ${PAX} -pe ${WRKSRC} ${DESTDIR}${PREFIX}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: Use the -pp option to pax(1) instead of -pe.",
		"WARN: Makefile:5: Use the -pp option to pax(1) instead of -pe.")
}

func (s *Suite) Test_SimpleCommandChecker_checkEchoN(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "ECHO", AtRunTime)
	t.SetUpTool("echo -n", "ECHO_N", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"do-install:",
		"\t${RUN} ${ECHO} -n 'Computing...'",
		"\t${RUN} ${ECHO_N} 'Computing...'",
		"\t${RUN} ${ECHO} 'Computing...'")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: Use ${ECHO_N} instead of \"echo -n\".")
}
