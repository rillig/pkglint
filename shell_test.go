package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestReShellToken(c *check.C) {
	re := `^(?:` + reShellToken + `)$`
	matches := check.NotNil
	doesntMatch := check.IsNil

	c.Check(match("", re), doesntMatch)
	c.Check(match("$var", re), matches)
	c.Check(match("$var$var", re), matches)
	c.Check(match("$var;;", re), doesntMatch) // More than one token
	c.Check(match("'single-quoted'", re), matches)
	c.Check(match("\"", re), doesntMatch)       // Incomplete string
	c.Check(match("'...'\"...\"", re), matches) // Mixed strings
	c.Check(match("\"...\"", re), matches)
	c.Check(match("`cat file`", re), matches)
	c.Check(match("${file%.c}.o", re), matches)
}

func (s *Suite) TestSplitIntoShellTokens_LineContinuation(c *check.C) {
	line := NewLine("fname", 10, "dummy", nil)

	words, rest := splitIntoShellTokens(line, "if true; then \\")

	c.Check(words, check.DeepEquals, []string{"if", "true", ";", "then"})
	c.Check(rest, equals, "\\")

	words, rest = splitIntoShellTokens(line, "pax -s /.*~$$//g")

	c.Check(words, check.DeepEquals, []string{"pax", "-s", "/.*~$$//g"})
	c.Check(rest, equals, "")
}

func (s *Suite) TestChecklineMkShellCommandLine(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.mk = s.NewMkLines("fname",
		"# dummy")
	shline := NewMkShellLine(G.mk.mklines[0])

	shline.checkShellCommandLine("@# Comment")

	c.Check(s.Output(), equals, "")

	shline.checkShellCommandLine("uname=`uname`; echo $$uname")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Unknown shell command \"uname\".\n"+
		"WARN: fname:1: Please switch to \"set -e\" mode before using a semicolon to separate commands.\n"+
		"WARN: fname:1: Unknown shell command \"echo\".\n"+
		"WARN: fname:1: Unquoted shell variable \"uname\".\n")

	G.globalData.tools = map[string]bool{"echo": true}
	G.globalData.predefinedTools = map[string]bool{"echo": true}
	G.mk = s.NewMkLines("fname",
		"# dummy")
	G.globalData.InitVartypes()

	shline.checkShellCommandLine("echo ${PKGNAME:Q}") // vucQuotPlain

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: PKGNAME may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${PKGNAME} here.\n")

	shline.checkShellCommandLine("echo \"${CFLAGS:Q}\"") // vucQuotDquot

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please don't use the :Q operator in double quotes.\n"+
		"WARN: fname:1: CFLAGS may not be used in this file.\n"+
		"WARN: fname:1: Please use ${CFLAGS:M*:Q} instead of ${CFLAGS:Q} and make sure the variable appears outside of any quoting characters.\n")

	shline.checkShellCommandLine("echo '${COMMENT:Q}'") // vucQuotSquot

	c.Check(s.Output(), equals, "WARN: fname:1: COMMENT may not be used in this file.\n")

	shline.checkShellCommandLine("echo $$@")

	c.Check(s.Output(), equals, "WARN: fname:1: The $@ shell variable should only be used in double quotes.\n")

	shline.checkShellCommandLine("echo \"$$\"") // As seen by make(1); the shell sees: echo $

	c.Check(s.Output(), equals, "WARN: fname:1: Unquoted $ or strange shell variable found.\n")

	shline.checkShellCommandLine("echo \"\\n\"") // As seen by make(1); the shell sees: echo "\n"

	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"\\\\n\" instead of \"\\n\".\n")

	shline.checkShellCommandLine("${RUN} for f in *.c; do echo $${f%.c}; done")

	c.Check(s.Output(), equals, "")

	// Based on mail/thunderbird/Makefile, rev. 1.159
	shline.checkShellCommandLine("${RUN} subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Unknown shell command \"unzip\".\n"+
		"WARN: fname:1: The exitcode of the left-hand-side command of the pipe operator is ignored.\n"+
		"WARN: fname:1: Unknown shell command \"awk\".\n")

	// From mail/thunderbird/Makefile, rev. 1.159
	shline.checkShellCommandLine("" +
		"${RUN} for e in ${XPI_FILES}; do " +
		"  subdir=\"`${UNZIP_CMD} -c \"$$e\" install.rdf | awk '/^    <em:id>/ {sub(\".*<em:id>\",\"\");sub(\"</em:id>.*\",\"\");print;exit;}'`\" && " +
		"  ${MKDIR} \"${WRKDIR}/extensions/$$subdir\" && " +
		"  cd \"${WRKDIR}/extensions/$$subdir\" && " +
		"  ${UNZIP_CMD} -aqo $$e; " +
		"done")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: XPI_FILES is used but not defined. Spelling mistake?\n"+
		"WARN: fname:1: UNZIP_CMD is used but not defined. Spelling mistake?\n"+
		"WARN: fname:1: The exitcode of the left-hand-side command of the pipe operator is ignored.\n"+
		"WARN: fname:1: Unknown shell command \"awk\".\n"+
		"WARN: fname:1: MKDIR is used but not defined. Spelling mistake?\n"+
		"WARN: fname:1: Unknown shell command \"${MKDIR}\".\n"+
		"WARN: fname:1: UNZIP_CMD is used but not defined. Spelling mistake?\n"+
		"WARN: fname:1: Unquoted shell variable \"e\".\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_nofix(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.InitVartypes()
	s.RegisterTool("echo", "ECHO", false)
	G.mk = s.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	shline := NewMkShellLine(G.mk.mklines[0])

	c.Check(shline.line.raw[0].textnl, equals, "\techo ${PKGNAME:Q}\n")
	c.Check(shline.line.raw[0].lineno, equals, 1)

	shline.checkShellCommandLine("echo ${PKGNAME:Q}")

	c.Check(s.Output(), equals, ""+
		"NOTE: Makefile:1: The :Q operator isn't necessary for ${PKGNAME} here.\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_showAutofix(c *check.C) {
	s.UseCommandLine(c, "-Wall", "--show-autofix")
	G.globalData.InitVartypes()
	s.RegisterTool("echo", "ECHO", false)
	G.mk = s.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	shline := NewMkShellLine(G.mk.mklines[0])

	shline.checkShellCommandLine("echo ${PKGNAME:Q}")

	c.Check(s.Output(), equals, ""+
		"NOTE: Makefile:1: The :Q operator isn't necessary for ${PKGNAME} here.\n"+
		"NOTE: Makefile:1: Autofix: replacing \"${PKGNAME:Q}\" with \"${PKGNAME}\".\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_autofix(c *check.C) {
	s.UseCommandLine(c, "-Wall", "--autofix")
	G.globalData.InitVartypes()
	s.RegisterTool("echo", "ECHO", false)
	G.mk = s.NewMkLines("Makefile",
		"\techo ${PKGNAME:Q}")
	shline := NewMkShellLine(G.mk.mklines[0])

	shline.checkShellCommandLine("echo ${PKGNAME:Q}")

	c.Check(s.Output(), equals, ""+
		"NOTE: Makefile:1: Autofix: replacing \"${PKGNAME:Q}\" with \"${PKGNAME}\".\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_InternalError1(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.InitVartypes()
	G.mk = s.NewMkLines("fname",
		"# dummy")
	shline := NewMkShellLine(G.mk.mklines[0])

	// foobar="`echo \"foo   bar\"`"
	shline.checkShellCommandLine("foobar=\"`echo \\\"foo   bar\\\"`\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Unknown shell command \"echo\".\n"+
		"ERROR: fname:1: Internal pkglint error: MkShellLine.checkShellword state=plain, rest=\"\\\\foo\", shellword=\"\\\\foo\"\n"+
		"ERROR: fname:1: Internal pkglint error: MkShellLine.checkShellCommand state=continuation rest=\"\\\\\" shellcmd=\"echo \\\\foo   bar\\\\\"\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_DollarWithoutVariable(c *check.C) {
	G.globalData.InitVartypes()
	G.mk = s.NewMkLines("fname",
		"# dummy")
	shline := NewMkShellLine(G.mk.mklines[0])
	s.RegisterTool("pax", "PAX", false)
	G.mk.tools["pax"] = true

	shline.checkShellCommandLine("pax -rwpp -s /.*~$$//g . ${DESTDIR}${PREFIX}")

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestChecklineMkShellword(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.InitVartypes()
	shline := NewMkShellLine(NewMkLine(NewLine("fname", 1, "# dummy", nil)))

	c.Check(matches("${list}", `^`+reVarnameDirect+`$`), equals, false)

	shline.checkShellword("${${list}}", false)

	c.Check(s.Output(), equals, "")

	shline.checkShellword("\"$@\"", false)

	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"${.TARGET}\" instead of \"$@\".\n")

	shline.checkShellword("${COMMENT:Q}", true)

	c.Check(s.Output(), equals, "WARN: fname:1: COMMENT may not be used in this file.\n")

	shline.checkShellword("\"${DISTINFO_FILE:Q}\"", true)

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: DISTINFO_FILE may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${DISTINFO_FILE} here.\n")

	shline.checkShellword("embed${DISTINFO_FILE:Q}ded", true)

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: DISTINFO_FILE may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${DISTINFO_FILE} here.\n")

	shline.checkShellword("s,\\.,,", true)

	c.Check(s.Output(), equals, "")

	shline.checkShellword("\"s,\\.,,\"", true)

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestMkShellLine_CheckShellword_DollarWithoutVariable(c *check.C) {
	shline := NewMkShellLine(NewMkLine(NewLine("fname", 1, "# dummy", nil)))

	shline.checkShellword("/.*~$$//g", false) // Typical argument to pax(1).

	c.Check(s.Output(), equals, "")
}

func (s *Suite) TestShelltextContext_CheckCommandStart(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	s.RegisterTool("echo", "ECHO", true)
	G.mk = s.NewMkLines("fname",
		"# dummy")
	mkline := NewMkLine(NewLine("fname", 3, "# dummy", nil))

	mkline.checkText("echo \"hello, world\"")

	c.Check(s.Output(), equals, "")

	NewMkShellLine(mkline).checkShellCommandLine("echo \"hello, world\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:3: Please use \"${ECHO}\" instead of \"echo\".\n")
}

func (s *Suite) TestMkShellLine_checklineMkShelltext(c *check.C) {

	shline := NewMkShellLine(NewMkLine(NewLine("Makefile", 3, "# dummy", nil)))

	shline.checkShellCommandLine("for f in *.pl; do ${SED} s,@PREFIX@,${PREFIX}, < $f > $f.tmp && ${MV} $f.tmp $f; done")

	c.Check(s.Output(), equals, "NOTE: Makefile:3: Please use the SUBST framework instead of ${SED} and ${MV}.\n")

	shline.checkShellCommandLine("install -c manpage.1 ${PREFIX}/man/man1/manpage.1")

	c.Check(s.Output(), equals, "WARN: Makefile:3: Please use ${PKGMANDIR} instead of \"man\".\n")

	shline.checkShellCommandLine("cp init-script ${PREFIX}/etc/rc.d/service")

	c.Check(s.Output(), equals, "WARN: Makefile:3: Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.\n")
}

func (s *Suite) TestMkShellLine_checkCommandUse(c *check.C) {
	G.mk = s.NewMkLines("fname",
		"# dummy")
	G.mk.target = "do-install"

	shline := NewMkShellLine(NewMkLine(NewLine("fname", 1, "\tdummy", nil)))

	shline.checkCommandUse("sed")

	c.Check(s.Output(), equals, "WARN: fname:1: The shell command \"sed\" should not be used in the install phase.\n")

	shline.checkCommandUse("cp")

	c.Check(s.Output(), equals, "WARN: fname:1: ${CP} should not be used to install files.\n")
}

func (s *Suite) TestSplitIntoShellWords(c *check.C) {
	url := "http://registry.gimp.org/file/fix-ca.c?action=download&id=9884&file="

	words, rest := splitIntoShellTokens(dummyLine, url)

	c.Check(words, check.DeepEquals, []string{"http://registry.gimp.org/file/fix-ca.c?action=download", "&", "id=9884", "&", "file="})
	c.Check(rest, equals, "")

	words, rest = splitIntoShellWords(dummyLine, url)

	c.Check(words, check.DeepEquals, []string{"http://registry.gimp.org/file/fix-ca.c?action=download&id=9884&file="})
	c.Check(rest, equals, "")

	words, rest = splitIntoShellWords(dummyLine, "a b \"c  c  c\" d;;d;; \"e\"''`` 'rest")

	c.Check(words, check.DeepEquals, []string{"a", "b", "\"c  c  c\"", "d;;d;;", "\"e\"''``"})
	c.Check(rest, equals, "'rest")
}
