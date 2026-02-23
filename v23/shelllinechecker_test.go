package pkglint

import (
	"gopkg.in/check.v1"
	"strings"
)

// Before 2020-03-25, pkglint ran into a parse error since it didn't
// know that _ULIMIT_CMD brings its own semicolon.
func (s *Suite) Test_ShellLineChecker__skip_ULIMIT_CMD(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"pre-configure:",
		"\t${RUN} ${_ULIMIT_CMD} while :; do :; done")

	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker__RUN(c *check.C) {
	t := s.Init(c)
	t.SetUpTool("echo", "", AfterPrefsMk)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"pre-configure:",
		"\t${RUN} echo good",
		"\t@${RUN} echo bad",
		"\tcd build && ${RUN} echo bad",
		"\tcd ${RUN}",
	)

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: The shell command \"${RUN}\" should not be hidden.",
		"ERROR: Makefile:5: The expression \"${RUN}\" must only occur "+
			"at the beginning of a shell command line.",
		"WARN: Makefile:5: Unknown shell command \"${RUN}\".",
		"WARN: Makefile:5: Variable \"RUN\" is used but not defined.",
		"ERROR: Makefile:6: The expression \"${RUN}\" must only occur "+
			"at the beginning of a shell command line.",
	)
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("[", "TEST", AtRunTime)
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

	test("@# Comment",
		nil...)

	test("uname=`uname`; echo $$uname; echo; ${PREFIX}/bin/command",
		"WARN: filename.mk:1: Unknown shell command \"uname\".",
		"WARN: filename.mk:1: Switch to \"set -e\" mode "+
			"before using a semicolon (after \"uname=`uname`\") to separate commands.")

	test("echo ${PKGNAME:Q}", // EctxQuotPlain
		"NOTE: filename.mk:1: The :Q modifier isn't necessary for ${PKGNAME} here.")

	test("echo \"${CFLAGS:Q}\"", // EctxQuotDquot

		// ShellLineChecker.checkExprToken
		"WARN: filename.mk:1: The :Q modifier should not be used inside quotes.",

		// ShellLineChecker.checkExprToken
		//     MkLineChecker.CheckExpr
		//         MkExprChecker.checkQuoting
		"WARN: filename.mk:1: Use ${CFLAGS:M*:Q} instead of ${CFLAGS:Q} "+
			"and make sure the variable appears outside of any quoting characters.")

	test("echo '${COMMENT:Q}'", // EctxQuotSquot
		"WARN: filename.mk:1: The :Q modifier should not be used inside quotes.",
		"WARN: filename.mk:1: Move ${COMMENT:Q} outside of any quoting characters.")

	test("echo target=$@ exitcode=$$? '$$' \"\\$$\"",
		"WARN: filename.mk:1: Use \"${.TARGET}\" instead of \"$@\".",
		"WARN: filename.mk:1: The $? shell variable is often not available in \"set -e\" mode.")

	test("echo $$@",
		"WARN: filename.mk:1: The $@ shell variable should only be used in double quotes.")

	// No warning about a possibly missed variable name.
	// This occurs only rarely, and typically as part of a regular expression
	// where it is used intentionally.
	test("echo \"$$\"", // As seen by make(1); the shell sees: echo "$"
		nil...)

	test("echo \"\\n\"",
		nil...)

	test("${RUN} for f in *.c; do echo $${f%.c}; done",
		nil...)

	test("${RUN} set +x; echo $${variable+set}",
		nil...)

	// Based on mail/thunderbird/Makefile, rev. 1.159
	test("${RUN} subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"",
		"WARN: filename.mk:1: Double quotes inside backticks inside double quotes are error prone.",
		"WARN: filename.mk:1: The exitcode of \"unzip\" at the left of the | operator is ignored.")

	// From mail/thunderbird/Makefile, rev. 1.159
	test(""+
		"${RUN} for e in ${XPI_FILES}; do "+
		"  subdir=\"`${UNZIP_CMD} -c \"$$e\" install.rdf | "+
		""+"awk '/.../ {print;exit;}'`\" && "+
		"  ${MKDIR} \"${WRKDIR}/extensions/$$subdir\" && "+
		"  cd \"${WRKDIR}/extensions/$$subdir\" && "+
		"  ${UNZIP_CMD} -aqo $$e; "+
		"done",
		"WARN: filename.mk:1: Variable \"XPI_FILES\" is used but not defined.",
		"WARN: filename.mk:1: Double quotes inside backticks inside double quotes are error prone.",
		"WARN: filename.mk:1: The exitcode of \"${UNZIP_CMD}\" at the left of the | operator is ignored.")

	// From x11/wxGTK28/Makefile
	test(""+
		"set -e; cd ${WRKSRC}/locale; "+
		"for lang in *.po; do "+
		"  [ \"$${lang}\" = \"wxstd.po\" ] && continue; "+
		"  ${TOOLS_PATH.msgfmt} -c -o \"$${lang%.po}.mo\" \"$${lang}\"; "+
		"done",
		nil...)

	test("@cp from to",
		"WARN: filename.mk:1: The shell command \"cp\" should not be hidden.")

	test("-cp from to",
		"WARN: filename.mk:1: Using a leading \"-\" to suppress errors is deprecated.")

	test("-${MKDIR} deeply/nested/subdir",
		"WARN: filename.mk:1: Using a leading \"-\" to suppress errors is deprecated.")

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

	// A directory that is not found in the PLIST.
	test("${RUN} ${INSTALL_DATA_DIR} ${PREFIX}/share/other",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= share/other\" instead of \"${INSTALL_DATA_DIR}\".")

	pkg = nil

	// See PR 46570, item "1. It does not"
	// No warning about missing error checking ("set -e").
	test("for x in 1 2 3; do echo \"$$x\" || exit 1; done",
		nil...)
}

// TODO: Document in detail that strip is not a regular tool.
func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__strip(c *check.C) {
	t := s.Init(c)

	test := func(shellCommand string) {
		mklines := t.NewMkLines("filename.mk",
			"\t"+shellCommand)

		mklines.ForEach(func(mkline *MkLine) {
			ck := NewShellLineChecker(mklines, mkline)
			ck.CheckShellCommandLine(mkline.ShellCommand())
		})
	}

	test("${STRIP} executable")

	t.CheckOutputLines(
		"WARN: filename.mk:1: Unknown shell command \"${STRIP}\".",
		"WARN: filename.mk:1: Variable \"STRIP\" is used but not defined.")

	t.SetUpVartypes()

	test("${STRIP} executable")

	t.CheckOutputEmpty()
}

// After working a lot with usr.bin/make, I thought that lines containing
// the cd command would differ in behavior between compatibility mode and
// parallel mode.  But since pkgsrc does not support parallel mode and also
// actively warns when someone tries to run it in parallel mode, there is
// no point checking for chdir that might spill over to the next line.
// That will not happen in compat mode.
func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__chdir(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "", AfterPrefsMk)
	t.SetUpTool("sed", "", AfterPrefsMk)
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"pre-configure:",
		// This command is run in the current directory.
		"\techo command 1",
		// This chdir affects the remaining commands.
		// It might be possible to warn here about chdir.
		"\tcd ..",
		// In subshells, chdir is ok.
		"\t(cd ..)",
		// In pipes, chdir is ok.
		"\t{ cd .. && echo sender; } | { cd .. && sed s,sender,receiver; }",
		// The && operator does not run in a subshell.
		// It might be possible to warn here about chdir.
		"\tcd .. && echo",
		// The || operator does not run in a subshell.
		// It might be possible to warn here about chdir.
		"\tcd .. || echo",
		// The current directory of this command depends on the preceding
		// commands.
		"\techo command 2",
		// In the final command of a target, chdir is ok since there are
		// no further commands that could be affected.
		"\tcd ..")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:7: The exitcode of the command at the left of " +
			"the | operator is ignored.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__nofix(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("echo", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	ck := NewShellLineChecker(mklines, mklines.mklines[0])

	ck.CheckShellCommandLine("echo ${PKGNAME:Q}")

	t.CheckOutputLines(
		"NOTE: Makefile:1: The :Q modifier isn't necessary for ${PKGNAME} here.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__show_autofix(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--show-autofix")
	t.SetUpVartypes()
	t.SetUpTool("echo", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	ck := NewShellLineChecker(mklines, mklines.mklines[0])

	ck.CheckShellCommandLine("echo ${PKGNAME:Q}")

	t.CheckOutputLines(
		"NOTE: Makefile:1: The :Q modifier isn't necessary for ${PKGNAME} here.",
		"AUTOFIX: Makefile:1: Replacing \"${PKGNAME:Q}\" with \"${PKGNAME}\".")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__autofix(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	t.SetUpVartypes()
	t.SetUpTool("echo", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	ck := NewShellLineChecker(mklines, mklines.mklines[0])

	ck.CheckShellCommandLine("echo ${PKGNAME:Q}")

	t.CheckOutputLines(
		"AUTOFIX: Makefile:1: Replacing \"${PKGNAME:Q}\" with \"${PKGNAME}\".")

	// TODO: There should be a general way of testing a code in the three modes:
	//  default, --show-autofix, --autofix.
}

// TODO: Document the exact purpose of this test, or split it into useful tests.
func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__implementation(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("filename.mk",
		"# dummy")
	ck := NewShellLineChecker(mklines, mklines.mklines[0])

	// foobar="`echo \"foo   bar\"`"
	text := "foobar=\"`echo \\\"foo   bar\\\"`\""

	tokens, rest := splitIntoShellTokens(ck.mkline.Line, text)

	t.CheckDeepEquals(tokens, []string{text})
	t.CheckEquals(rest, "")

	mklines.ForEach(func(mkline *MkLine) { ck.CheckWord(text, false, RunTime) })

	t.CheckOutputLines(
		"WARN: filename.mk:1: Unknown shell command \"echo\".")

	mklines.ForEach(func(mkline *MkLine) { ck.CheckShellCommandLine(text) })

	// No parse errors
	t.CheckOutputLines(
		"WARN: filename.mk:1: Unknown shell command \"echo\".")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__dollar_without_variable(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("pax", "", AtRunTime)
	mklines := t.NewMkLines("filename.mk",
		"# dummy")
	ck := NewShellLineChecker(mklines, mklines.mklines[0])

	ck.CheckShellCommandLine("pax -rwpp -s /.*~$$//g . ${DESTDIR}${PREFIX}")

	t.CheckOutputLines(
		"WARN: filename.mk:1: Substitution commands like \"/.*~$$//g\" should always be quoted.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__echo(c *check.C) {
	t := s.Init(c)

	echo := t.SetUpTool("echo", "ECHO", AtRunTime)
	echo.MustUseVarForm = true
	mklines := t.NewMkLines("filename.mk",
		"# dummy")
	mkline := t.NewMkLine("filename.mk", 3, "# dummy")

	MkLineChecker{mklines, mkline}.checkText("echo \"hello, world\"")

	t.CheckOutputEmpty()

	NewShellLineChecker(mklines, mkline).CheckShellCommandLine("echo \"hello, world\"")

	t.CheckOutputLines(
		"WARN: filename.mk:3: Use \"${ECHO}\" instead of \"echo\".")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__shell_variables(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("install", "INSTALL", AtRunTime)
	t.SetUpTool("cp", "CP", AtRunTime)
	t.SetUpTool("mv", "MV", AtRunTime)
	t.SetUpTool("sed", "SED", AtRunTime)
	text := "for f in *.pl; do ${SED} s,@PREFIX@,${PREFIX}, < $f > $f.tmp && ${MV} $f.tmp $f; done"
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"\t"+text)

	ck := NewShellLineChecker(mklines, mklines.mklines[2])
	ck.CheckShellCommandLine(text)

	t.CheckOutputLines(
		// TODO: Avoid these duplicate diagnostics.
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"NOTE: Makefile:3: Use the SUBST framework instead of ${SED} and ${MV}.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: Variable \"f\" is used but not defined.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.",
		"WARN: Makefile:3: $f is ambiguous. Use ${f} if you mean a Make variable or $$f if you mean a shell variable.")

	ck.CheckShellCommandLine("install -c manpage.1 ${PREFIX}/man/man1/manpage.1")

	t.CheckOutputLines(
		"WARN: Makefile:3: Use ${PKGMANDIR} instead of \"man\".")

	ck.CheckShellCommandLine("cp init-script ${PREFIX}/etc/rc.d/service")

	t.CheckOutputLines(
		"WARN: Makefile:3: Use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__sed_and_mv(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("sed", "SED", AtRunTime)
	t.SetUpTool("mv", "MV", AtRunTime)
	ck := t.NewShellLineChecker("\t${RUN} ${SED} 's,#,// comment:,g' filename > filename.tmp; ${MV} filename.tmp filename")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	t.CheckOutputLines(
		"NOTE: filename.mk:1: Use the SUBST framework instead of ${SED} and ${MV}.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__sed_and_mv_explained(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--explain")
	t.SetUpVartypes()
	t.SetUpTool("sed", "SED", AtRunTime)
	t.SetUpTool("mv", "MV", AtRunTime)
	ck := t.NewShellLineChecker("\t${RUN} ${SED} 's,#,// comment:,g' filename > filename.tmp; ${MV} filename.tmp filename")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	t.CheckOutputLines(
		"NOTE: filename.mk:1: Use the SUBST framework instead of ${SED} and ${MV}.",
		"",
		"\tUsing the SUBST framework instead of explicit commands is easier to",
		"\tunderstand, since all the complexity of using sed and mv is hidden",
		"\tbehind the scenes.",
		"",
		sprintf("\tRun %q for more information.", bmakeHelp("subst")),
		"",
		"\tWhen migrating to the SUBST framework, pay attention to \"#\"",
		"\tcharacters. In shell commands, make(1) does not interpret them as",
		"\tcomment character, but in variable assignments it does. Therefore,",
		"\tinstead of the shell command",
		"",
		"\t\tsed -e 's,#define foo,,'",
		"",
		"\tyou need to write",
		"",
		"\t\tSUBST_SED.foo+=\t's,\\#define foo,,'",
		"")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__sed_and_mv_autofix_explained(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--explain", "--autofix")
	t.SetUpVartypes()
	t.SetUpTool("sed", "SED", AtRunTime)
	t.SetUpTool("mv", "MV", AtRunTime)
	ck := t.NewShellLineChecker("\t${RUN} ${SED} 's,#,// comment:,g' filename > filename.tmp; ${MV} filename.tmp filename")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	// Only ever output an explanation if there's a corresponding diagnostic.
	// Even if Explain is called twice in a row.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__subshell(c *check.C) {
	t := s.Init(c)

	ck := t.NewShellLineChecker("\t${RUN} uname=$$(uname)")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	// Up to 2020-05-09, pkglint had warned that $(...) were not portable
	// enough. The shell used in devel/bmake can handle these subshell
	// command substitutions though.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__install_dir(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	ck := t.NewShellLineChecker("\t${RUN} ${INSTALL_DATA_DIR} ${DESTDIR}${PREFIX}/dir1 ${DESTDIR}${PREFIX}/dir2")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	t.CheckOutputLines(
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir1\" instead of \"${INSTALL_DATA_DIR}\".",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir2\" instead of \"${INSTALL_DATA_DIR}\".",
		"WARN: filename.mk:1: The INSTALL_*_DIR commands can only handle one directory at a time.")

	ck.CheckShellCommandLine("${INSTALL_DATA_DIR} -d -m 0755 ${DESTDIR}${PREFIX}/share/examples/gdchart")

	// No warning about multiple directories, since 0755 is an option, not an argument.
	t.CheckOutputLines(
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= share/examples/gdchart\" instead of \"${INSTALL_DATA_DIR}\".")

	ck.CheckShellCommandLine("${INSTALL_DATA_DIR} -d -m 0755 ${DESTDIR}${PREFIX}/dir1 ${PREFIX}/dir2")

	t.CheckOutputLines(
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir1\" instead of \"${INSTALL_DATA_DIR}\".",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir2\" instead of \"${INSTALL_DATA_DIR}\".",
		"WARN: filename.mk:1: The INSTALL_*_DIR commands can only handle one directory at a time.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__install_option_d(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	ck := t.NewShellLineChecker("\t${RUN} ${INSTALL} -d ${DESTDIR}${PREFIX}/dir1 ${DESTDIR}${PREFIX}/dir2")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	t.CheckOutputLines(
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir1\" instead of \"${INSTALL} -d\".",
		"NOTE: filename.mk:1: You can use \"INSTALLATION_DIRS+= dir2\" instead of \"${INSTALL} -d\".")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommandLine__trailing_semicolon(c *check.C) {
	t := s.Init(c)
	t.SetUpTool("mkdir", "", AfterPrefsMk)
	t.SetUpTool("find", "", AfterPrefsMk)

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"do-configure:",
		"\tmkdir -p dirname;",
		"\tfind . -name '*.h' -exec {} \\;")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: Makefile:3: A trailing semicolon " +
			"at the end of a shell command line is redundant.")
}

func (s *Suite) Test_ShellLineChecker_checkHiddenAndSuppress(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("echo", "ECHO", AtRunTime)
	t.SetUpTool("ls", "LS", AtRunTime)
	t.SetUpTool("mkdir", "MKDIR", AtRunTime)
	t.SetUpTool("printf", "PRINTF", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"show-all-targets: .PHONY",
		"\t@echo 'hello'",
		"\t@ls -l",
		"",
		"anything-message: .PHONY",
		"\t@echo 'may be hidden'",
		"\t@ls 'may be hidden'",
		"",
		"pre-configure:",
		"\t@",
		"\t@mkdir ${WRKSRC}",
		"\t@${DELAYED_ERROR_MSG} 'ok'",
		"\t@${DELAYED_WARNING_MSG} 'ok'",
		"\t@${DO_NADA} 'ok'",
		"\t@${ECHO} 'ok'",
		"\t@echo 'ok'",
		"\t@${ECHO_N} 'ok'",
		"\t@${ECHO_MSG} 'ok'",
		"\t@${ERROR_CAT} 'ok'",
		"\t@${ERROR_MSG} 'ok'",
		"\t@${FAIL_MSG} 'ok'",
		"\t@${INFO_MSG} 'ok'",
		"\t@${PHASE_MSG} 'ok'",
		"\t@${PRINTF} 'ok'",
		"\t@printf 'ok'",
		"\t@${SHCOMMENT} 'ok'",
		"\t@${STEP_MSG} 'ok'",
		"\t@${WARNING_CAT} 'ok'",
		"\t@${WARNING_MSG} 'ok'")

	mklines.Check()

	// No warning about the hidden ls since the target names start
	// with "show-" or end with "-message".
	t.CheckOutputLines(
		"WARN: Makefile:13: The shell command \"mkdir\" should not be hidden.")
}

func (s *Suite) Test_ShellLineChecker_checkHiddenAndSuppress__no_tracing(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("ls", "LS", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"pre-configure:",
		"\t@ls -l")
	t.DisableTracing()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: The shell command \"ls\" should not be hidden.")
}

func (s *Suite) Test_ShellLineChecker_CheckWord(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	test := func(shellWord string, checkQuoting bool, diagnostics ...string) {
		// See MkExprChecker.checkUndefined and MkAssignChecker.checkLeftNotUsed.
		ck := t.NewShellLineChecker("\t echo " + shellWord)
		ck.CheckWord(shellWord, checkQuoting, RunTime)
		t.CheckOutput(diagnostics)
	}

	// No warning for the outer variable since it is completely indirect.
	// The inner variable ${list} must still be defined, though.
	test("${${list}}", false,
		"WARN: filename.mk:1: Variable \"list\" is used but not defined.")

	// No warning for variables that are partly indirect.
	// TODO: Why not?
	test("${SED_FILE.${id}}", false,
		"WARN: filename.mk:1: Variable \"id\" is used but not defined.")

	// TODO: Since $@ refers to ${.TARGET} and not sh.argv, there is no point in checking for quotes.
	//  The corresponding code in ShellLineChecker.CheckWord should be removed.
	// TODO: Having the same tests for $$@ would be much more interesting.

	// The unquoted $@ takes a different code path in pkglint than the quoted $@.
	test("$@", false,
		"WARN: filename.mk:1: Use \"${.TARGET}\" instead of \"$@\".")

	// When $@ appears as part of a shell token, it takes another code path in pkglint.
	test("-$@-", false,
		"WARN: filename.mk:1: Use \"${.TARGET}\" instead of \"$@\".")

	// The unquoted $@ takes a different code path in pkglint than the quoted $@.
	test("\"$@\"", false,
		"WARN: filename.mk:1: Use \"${.TARGET}\" instead of \"$@\".")

	test("${COMMENT:Q}", true,
		nil...)

	test("\"${DISTINFO_FILE:Q}\"", true,
		"NOTE: filename.mk:1: The :Q modifier isn't necessary for ${DISTINFO_FILE} here.")

	test("embed${DISTINFO_FILE:Q}ded", true,
		"NOTE: filename.mk:1: The :Q modifier isn't necessary for ${DISTINFO_FILE} here.")

	test("s,\\.,,", true,
		nil...)

	test("\"s,\\.,,\"", true,
		nil...)
}

func (s *Suite) Test_ShellLineChecker_CheckWord__dollar_without_variable(c *check.C) {
	t := s.Init(c)

	ck := t.NewShellLineChecker("# dummy")

	ck.CheckWord("/.*~$$//g", false, RunTime) // Typical argument to pax(1).

	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_CheckWord__backslash_plus(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("find", "FIND", AtRunTime)
	ck := t.NewShellLineChecker("\tfind . -exec rm -rf {} \\+")

	ck.CheckShellCommandLine(ck.mkline.ShellCommand())

	// A backslash before any other character than " \ ` is discarded by the parser.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_CheckWord__squot_dollar(c *check.C) {
	t := s.Init(c)

	ck := t.NewShellLineChecker("\t'$")

	ck.CheckWord(ck.mkline.ShellCommand(), false, RunTime)

	// XXX: Should be parsed correctly. Make passes the dollar through (probably),
	//  and the shell parser should complain about the unfinished string literal.
	t.CheckOutputLines(
		"WARN: filename.mk:1: Internal pkglint error in ShTokenizer.ShAtom at \"$\" (quoting=s).",
		"WARN: filename.mk:1: Internal pkglint error in ShellLine.CheckWord at \"'$\" (quoting=s), rest: $")
}

func (s *Suite) Test_ShellLineChecker_CheckWord__dquot_dollar(c *check.C) {
	t := s.Init(c)

	ck := t.NewShellLineChecker("\t\"$")

	ck.CheckWord(ck.mkline.ShellCommand(), false, RunTime)

	// XXX: Make consumes the dollar silently.
	//  This could be worth another pkglint warning.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_CheckWord__PKGMANDIR(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("chat/ircII/Makefile",
		MkCvsID,
		"CONFIGURE_ARGS+=--mandir=${DESTDIR}${PREFIX}/man",
		"CONFIGURE_ARGS+=--mandir=${DESTDIR}${PREFIX}/${PKGMANDIR}")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: chat/ircII/Makefile:2: Use ${PKGMANDIR} instead of \"man\".",
		"NOTE: chat/ircII/Makefile:2: This variable value should be aligned to column 25 instead of 17.",
		"NOTE: chat/ircII/Makefile:3: This variable value should be aligned to column 25 instead of 17.")
}

func (s *Suite) Test_ShellLineChecker_CheckWord__empty(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"JAVA_CLASSPATH=\t# empty")

	mklines.Check()

	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_checkWordQuoting(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("grep", "GREP", AtRunTime)

	test := func(input string, diagnostics ...string) {
		mklines := t.NewMkLines("module.mk",
			"\t"+input)
		ck := NewShellLineChecker(mklines, mklines.mklines[0])

		ck.checkWordQuoting(ck.mkline.ShellCommand(), true, RunTime)

		t.CheckOutput(diagnostics)
	}

	test(
		"socklen=`${GREP} 'expr' ${WRKSRC}/config.h`",
		nil...)

	test(
		"s,$$from,$$to,",
		"WARN: module.mk:1: Unquoted shell variable \"from\".",
		"WARN: module.mk:1: Unquoted shell variable \"to\".")

	// This variable is typically defined by GNU Configure,
	// which cannot handle directories with special characters.
	// Therefore, using it unquoted is considered safe.
	test(
		"${PREFIX}/$$bindir/program",
		nil...)

	test(
		"$$@",
		"WARN: module.mk:1: The $@ shell variable should only be used in double quotes.")

	// TODO: Add separate tests for "set +e" and "set -e".
	test(
		"$$?",
		"WARN: module.mk:1: The $? shell variable is often not available in \"set -e\" mode.")

	test(
		"$$(cat /bin/true)",
		nil...)

	test(
		"\"$$\"",
		nil...)

	test(
		"$$$$",
		nil...)

	test(
		"``",
		nil...)
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommand__subshell(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "ECHO", AtRunTime)
	t.SetUpTool("expr", "EXPR", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"pre-configure:",
		"\t@(echo ok)",
		"\techo $$(uname -r); echo $$(expr 4 '*' $$(echo 1024))",
		"\t@(echo nb$$(uname -r) $$(${EXPR} 4 \\* $$(echo 1024)))")

	mklines.Check()

	// XXX: Fix the parse errors (nested subshells).
	// XXX: Fix the duplicate diagnostic in line 6.
	// TODO: "(" is not a shell command, it's an operator.
	t.CheckOutputLines(
		"WARN: Makefile:4: The shell command \"(\" should not be hidden.",
		"WARN: Makefile:5: Internal pkglint error in ShTokenizer.ShAtom at \"$$(echo 1024))\" (quoting=S).",
		"WARN: Makefile:5: Invoking subshells via $(...) is not portable enough.",
		"WARN: Makefile:6: Internal pkglint error in ShTokenizer.ShAtom at \"$$(echo 1024)))\" (quoting=S).",
		"WARN: Makefile:6: The shell command \"(\" should not be hidden.",
		"WARN: Makefile:6: Internal pkglint error in ShTokenizer.ShAtom at \"$$(echo 1024)))\" (quoting=S).",
		"WARN: Makefile:6: Invoking subshells via $(...) is not portable enough.")
}

func (s *Suite) Test_ShellLineChecker_CheckShellCommand__case_patterns_from_variable(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"",
		"pre-configure:",
		"\tcase $$file in ${CHECK_PERMS_SKIP:@pattern@${pattern}) ;;@} *) continue; esac")

	mklines.Check()

	// TODO: Ensure that the shell word is really only one expression.
	// TODO: Ensure that the last modifier is :@@@.
	// TODO: Ensure that the replacement is a well-formed case-item.
	// TODO: Ensure that the replacement contains ";;" as the last shell token.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_ShellLineChecker_checkSetE__simple_commands(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("rm", "", AtRunTime)
	t.SetUpTool("touch", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"pre-configure:",
		"\techo 1; echo 2; echo 3",
		"\techo 1; touch file; rm file",
		"\techo 1; var=value; echo 3")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:4: Switch to \"set -e\" mode before using a semicolon " +
			"(after \"touch file\") to separate commands.")
}

func (s *Suite) Test_ShellLineChecker_checkSetE__compound_commands(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("touch", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"pre-configure:",
		"\ttouch file; for f in file; do echo \"$$f\"; done",
		"\tfor f in file; do echo \"$$f\"; done; touch file",
		"\ttouch 1; touch 2; touch 3; touch 4")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"touch file\") to separate commands.",
		"WARN: Makefile:5: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"touch 1\") to separate commands.")
}

func (s *Suite) Test_ShellLineChecker_checkSetE__no_tracing(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("touch", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		MkCvsID,
		"pre-configure:",
		"\ttouch 1; touch 2")
	t.DisableTracing()

	mklines.Check()

	t.CheckOutputLines(
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon " +
			"(after \"touch 1\") to separate commands.")
}

func (s *Suite) Test_ShellLineChecker_checkPipeExitcode(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("cat", "", AtRunTime)
	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("printf", "", AtRunTime)
	t.SetUpTool("sed", "", AtRunTime)
	t.SetUpTool("right-side", "", AtRunTime)
	mklines := t.NewMkLines("Makefile",
		"\t echo | right-side",
		"\t sed s,s,s, | right-side",
		"\t printf | sed s,s,s, | right-side ",
		"\t cat | right-side",
		"\t cat | echo | right-side",
		"\t echo | cat | right-side",
		"\t sed s,s,s, filename | right-side",
		"\t sed s,s,s < input | right-side",
		"\t ./unknown | right-side",
		"\t var=value | right-side",
		"\t if :; then :; fi | right-side",
		"\t var=`cat file` | right-side")

	for _, mkline := range mklines.mklines {
		ck := NewShellLineChecker(mklines, mkline)
		ck.CheckShellCommandLine(mkline.ShellCommand())
	}

	t.CheckOutputLines(
		"WARN: Makefile:4: The exitcode of \"cat\" at the left of the | operator is ignored.",
		"WARN: Makefile:5: The exitcode of \"cat\" at the left of the | operator is ignored.",
		"WARN: Makefile:6: The exitcode of \"cat\" at the left of the | operator is ignored.",
		"WARN: Makefile:7: The exitcode of \"sed\" at the left of the | operator is ignored.",
		"WARN: Makefile:8: The exitcode of \"sed\" at the left of the | operator is ignored.",
		"WARN: Makefile:9: The exitcode of \"./unknown\" at the left of the | operator is ignored.",
		"WARN: Makefile:11: The exitcode of the command at the left of the | operator is ignored.",
		"WARN: Makefile:12: The exitcode of the command at the left of the | operator is ignored.")
}

func (s *Suite) Test_ShellLineChecker_canFail(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("basename", "", AtRunTime)
	t.SetUpTool("dirname", "", AtRunTime)
	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("env", "", AtRunTime)
	t.SetUpTool("ggrep", "", AtRunTime)
	t.SetUpTool("grep", "GREP", AtRunTime)
	t.SetUpTool("sed", "", AtRunTime)
	t.SetUpTool("gsed", "", AtRunTime)
	t.SetUpTool("touch", "", AtRunTime)
	t.SetUpTool("tr", "tr", AtRunTime)
	t.SetUpTool("true", "TRUE", AtRunTime)

	test := func(cmd string, diagnostics ...string) {
		mklines := t.NewMkLines("Makefile",
			MkCvsID,
			"pre-configure:",
			"\t"+cmd+" ; echo 'done.'")

		mklines.Check()

		t.CheckOutput(diagnostics)
	}

	test("socklen=`${GREP} 'expr' ${WRKSRC}/config.h`",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"socklen=`${GREP} 'expr' ${WRKSRC}/config.h`\") to separate commands.")

	test("socklen=`${GREP} 'expr' ${WRKSRC}/config.h || ${TRUE}`",
		nil...)

	test("socklen=$$(expr 16)",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"socklen=$$(expr 16)\") to separate commands.")

	test("socklen=$$(expr 16 || true)",
		nil...)

	test("socklen=$$(expr 16 || ${TRUE})",
		nil...)

	test("${ECHO_MSG} \"Message\"",
		nil...)

	test("${PHASE_MSG} \"Message\"",
		nil...)

	test("${STEP_MSG} \"Message\"",
		nil...)

	test("${INFO_MSG} \"Message\"",
		nil...)

	test("${WARNING_MSG} \"Message\"",
		nil...)

	test("${ERROR_MSG} \"Message\"",
		nil...)

	test("${WARNING_CAT} \"Message\"",
		nil...)

	test("${ERROR_CAT} \"Message\"",
		nil...)

	test("${DO_NADA} \"Message\"",
		nil...)

	test("${FAIL_MSG} \"Failure\"",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"${FAIL_MSG} \\\"Failure\\\"\") to separate commands.")

	test("set -x",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"set -x\") to separate commands.")

	test("echo 'input' | sed -e s,in,out,",
		nil...)

	test("sed -e s,in,out,",
		nil...)

	test("sed s,in,out,",
		nil...)

	test("gsed -e s,in,out,",
		nil...)

	test("gsed s,in,out,",
		nil...)

	test("gsed s,in,out, filename",
		"WARN: Makefile:3: Switch to \"set -e\" mode "+
			"before using a semicolon (after \"gsed s,in,out, filename\") "+
			"to separate commands.")

	test("ggrep input",
		nil...)

	test("ggrep pattern file...",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"ggrep pattern file...\") to separate commands.")

	test("grep input",
		nil...)

	test("grep pattern file...",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"grep pattern file...\") to separate commands.")

	test("touch file",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"touch file\") to separate commands.")

	test("echo 'starting'",
		nil...)

	test("echo 'logging' > log",
		"WARN: Makefile:3: Switch to \"set -e\" mode before using a semicolon "+
			"(after \"echo 'logging'\") to separate commands.")

	test("echo 'to stderr' 1>&2",
		nil...)

	test("echo 'hello' | tr -d 'aeiou'",
		nil...)

	test("env | grep '^PATH='",
		nil...)

	test("basename dir/file",
		nil...)

	test("dirname dir/file",
		nil...)

	test("tr A-Z a-z",
		nil...)
}

func (s *Suite) Test_ShellLineChecker_unescapeBackticks__unfinished(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"pre-configure:",
		"\t`${VAR}",      // Error in first shell word
		"\techo `${VAR}") // Error after first shell word

	// Breakpoint in ShellLine.CheckShellCommand
	// Breakpoint in ShellLine.CheckToken
	// Breakpoint in ShellLine.unescapeBackticks
	mklines.Check()

	t.CheckOutputLines(
		"WARN: filename.mk:4: Pkglint ShellLine.CheckShellCommand: splitIntoShellTokens couldn't parse \"`${VAR}\"",
		"WARN: filename.mk:5: Pkglint ShellLine.CheckShellCommand: splitIntoShellTokens couldn't parse \"`${VAR}\"")
}

func (s *Suite) Test_ShellLineChecker_unescapeBackticks__unfinished_direct(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("dummy.mk",
		MkCvsID,
		"\t# shell command")

	// This call is unrealistic. It doesn't happen in practice, and this
	// direct, forcing test is only to reach the code coverage.
	atoms := []*ShAtom{
		NewShAtom(shtText, "`", shqBackt)}
	NewShellLineChecker(mklines, mklines.mklines[1]).
		unescapeBackticks(&atoms, shqBackt)

	t.CheckOutputLines(
		"ERROR: dummy.mk:2: Unfinished backticks after \"\".")
}

func (s *Suite) Test_ShellLineChecker_unescapeBackticks(c *check.C) {
	t := s.Init(c)

	test := func(input string, expectedOutput string, expectedRest string, diagnostics ...string) {
		ck := t.NewShellLineChecker("# dummy")

		tok := NewShTokenizer(nil, input)
		atoms := tok.ShAtoms()

		// Set up the correct quoting mode for the test by skipping
		// uninteresting atoms at the beginning.
		q := shqPlain
		for atoms[0].MkText != "`" {
			q = atoms[0].Quoting
			atoms = atoms[1:]
		}
		t.CheckEquals(tok.Rest(), "")

		backtCommand := ck.unescapeBackticks(&atoms, q)

		var actualRest strings.Builder
		for _, atom := range atoms {
			actualRest.WriteString(atom.MkText)
		}

		t.CheckEquals(backtCommand, expectedOutput)
		t.CheckEquals(actualRest.String(), expectedRest)
		t.CheckOutput(diagnostics)
	}

	test("`echo`end", "echo", "end")
	test("`echo $$var`end", "echo $$var", "end")
	test("``end", "", "end")
	test("`echo \"hello\"`end", "echo \"hello\"", "end")
	test("`echo 'hello'`end", "echo 'hello'", "end")
	test("`echo '\\\\\\\\'`end", "echo '\\\\'", "end")

	// Only the characters " $ ` \ are unescaped. All others stay the same.
	test("`echo '\\n'`end", "echo '\\n'", "end",
		// TODO: Add more details regarding which backslash is meant.
		"WARN: filename.mk:1: Backslashes should be doubled inside backticks.")
	test("\tsocklen=`${GREP} 'expr' ${WRKSRC}/config.h`", "${GREP} 'expr' ${WRKSRC}/config.h", "")

	// The 2xx test cases are in shqDquot mode.

	test("\"`echo`\"", "echo", "\"")
	test("\"`echo \"\"`\"", "echo \"\"", "\"",
		"WARN: filename.mk:1: Double quotes inside backticks inside double quotes are error prone.")

	// varname="`echo \"one   two\" "\ " "three"`"
	test(
		"varname=\"`echo \\\"one   two\\\" \"\\ \" \"three\"`\"",
		"echo \"one   two\" \"\\ \" \"three\"",
		"\"",

		// TODO: Add more details regarding which backslash and backtick is meant.
		"WARN: filename.mk:1: Backslashes should be doubled inside backticks.",
		"WARN: filename.mk:1: Double quotes inside backticks inside double quotes are error prone.",
		"WARN: filename.mk:1: Double quotes inside backticks inside double quotes are error prone.")

	// The inner shell command in the backticks is malformed since it
	// contains an unpaired backtick.
	test("`echo \\``rest", "echo `", "rest")

	// Enclosing the inner backtick in single quotes makes it valid.
	test("`echo '\\`'`rest", "echo '`'", "rest")

	test("`echo \\$$var`rest", "echo $$var", "rest")
}

func (s *Suite) Test_ShellLineChecker_unescapeBackticks__dquotBacktDquot(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "", AtRunTime)
	mklines := t.NewMkLines("dummy.mk",
		MkCvsID,
		"\t var=\"`echo \"\"`\"")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: dummy.mk:2: Double quotes inside backticks inside double quotes are error prone.")
}

func (s *Suite) Test_ShellLineChecker_checkShExprPlain__default_warning_level(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine( /* none */ )
	t.SetUpVartypes()
	t.SetUpTool("echo", "", AtRunTime)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"CONFIGURE_ARGS+=\techo $$@ $$var",
		"",
		"pre-configure:",
		"\techo $$@ $$var")

	mklines.Check()

	// Using $@ outside of double quotes is so obviously wrong that
	// the warning is issued by default.
	t.CheckOutputLines(
		"WARN: filename.mk:2: The $@ shell variable should only be used in double quotes.",
		"WARN: filename.mk:5: The $@ shell variable should only be used in double quotes.")
}

func (s *Suite) Test_ShellLineChecker_checkShExprPlain__Wall(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("echo", "", AtRunTime)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"CONFIGURE_ARGS+=\techo $$@ $$var",
		"",
		"pre-configure:",
		"\techo $$@ $$var")

	mklines.Check()

	// XXX: It is inconsistent that the check for unquoted shell
	//  variables is enabled for CONFIGURE_ARGS (where shell variables
	//  don't make sense at all) but not for real shell commands.
	t.CheckOutputLines(
		"WARN: filename.mk:2: The $@ shell variable should only be used in double quotes.",
		"WARN: filename.mk:2: Unquoted shell variable \"var\".",
		"WARN: filename.mk:5: The $@ shell variable should only be used in double quotes.")
}

func (s *Suite) Test_ShellLineChecker_checkShExprPlain__dollarQuestion(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("tool", "", AtRunTime)

	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		"do-configure:",
		"\tif tool $$? != 0; then \\",
		"\t\ttool bad; \\",
		"\tfi",
	)

	mklines.Check()

	// FIXME: Only warn if there is actually a "set -e" command around.
	t.CheckOutputLines(
		"WARN: filename.mk:4--6: The $? shell variable is often not available in \"set -e\" mode.")
}

func (s *Suite) Test_ShellLineChecker_variableNeedsQuoting(c *check.C) {
	t := s.Init(c)

	test := func(shVarname string, expected bool) {
		t.CheckEquals((*ShellLineChecker).variableNeedsQuoting(nil, shVarname), expected)
	}

	test("#", false) // A length is always an integer.
	test("?", false) // The exit code is always an integer.
	test("$", false) // The PID is always an integer.

	// In most cases, file and directory names don't contain special characters,
	// and if they do, the package will probably not build. Therefore, pkglint
	// doesn't require them to be quoted, but doing so does not hurt.
	test("d", false)    // Typically used for directories.
	test("f", false)    // Typically used for files.
	test("i", false)    // Typically used for literal values without special characters.
	test("id", false)   // Identifiers usually don't use special characters.
	test("dir", false)  // See d above.
	test("file", false) // See f above.
	test("src", false)  // Typically used when copying files or directories.
	test("dst", false)  // Typically used when copying files or directories.

	test("bindir", false) // A typical GNU-style directory.
	test("mandir", false) // A typical GNU-style directory.
	test("prefix", false) //

	test("bindirs", true) // A list of directories is typically separated by spaces.
	test("var", true)     // Other variables are unknown, so they should be quoted.
	test("0", true)       // The program name may contain special characters when given as full path.
	test("1", true)       // Command line arguments can be arbitrary strings.
	test("comment", true) // Comments can be arbitrary strings.
}

func (s *Suite) Test_ShellLineChecker_variableNeedsQuoting__integration(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("cp", "", AtRunTime)
	mklines := t.NewMkLines("filename.mk",
		MkCvsID,
		"",
		// It's a bit silly to use shell variables in CONFIGURE_ARGS,
		// but as of January 2019 that's the only way to run ShellLine.variableNeedsQuoting.
		"CONFIGURE_ARGS+=\t; cp $$dir $$\\# $$target",
		"pre-configure:",
		"\tcp $$dir $$\\# $$target")

	mklines.Check()

	// As of January 2019, the quoting check is disabled for real shell commands.
	// See ShellLine.CheckShellCommand, spc.checkWord.
	t.CheckOutputLines(
		"WARN: filename.mk:3: Unquoted shell variable \"target\".")
}

func (s *Suite) Test_ShellLineChecker_checkMultiLineComment(c *check.C) {
	t := s.Init(c)

	t.SetUpTool("echo", "", AtRunTime)
	t.SetUpTool("sed", "", AtRunTime)
	t.SetUpVartypes()

	test := func(lines ...string) {
		i := 0
		for ; i < len(lines) && hasPrefix(lines[i], "\t"); i++ {
		}

		mklines := t.SetUpFileMkLines("Makefile",
			append([]string{MkCvsID, "pre-build:"},
				lines[:i]...)...)

		mklines.Check()

		t.CheckOutput(lines[i:])
	}

	// The comment can start at the beginning of a follow-up line.
	test(
		"\techo first; \\",
		"\t# comment at the beginning of a command \\",
		"\techo \"hello\"",

		"WARN: ~/Makefile:4: "+
			"The shell comment does not stop at the end of this line.")

	// The comment can start at the beginning of a simple command.
	test(
		"\techo first; # comment at the beginning of a command \\",
		"\techo \"hello\"",

		"WARN: ~/Makefile:3: "+
			"The shell comment does not stop at the end of this line.")

	// The comment can start at a word in the middle of a command.
	test(
		"\techo # comment starts inside a command \\",
		"\techo \"hello\"",

		"WARN: ~/Makefile:3: "+
			"The shell comment does not stop at the end of this line.")

	// If the comment starts in the last line, there's no further
	// line that might be commented out accidentally.
	test(
		"\techo 'first line'; \\",
		"\t# comment in last line")

	// If there's a shell token that extends over several lines,
	// that's unusual enough that pkglint refuses to check this.
	test(
		"\techo 'before \\",
		"\t\tafter'; \\",
		"\t# comment \\",
		"\techo 'still a comment'")

	test(
		"\tsed -e s#@PREFIX@#${PREFIX}#g \\",
		"\tfilename",

		"WARN: ~/Makefile:3--4: Substitution commands like "+
			"\"s#@PREFIX@#${PREFIX}#g\" should always be quoted.")
}
