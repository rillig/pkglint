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
	c.Check(match("$var;;", re), doesntMatch) // More than one shellword
	c.Check(match("'single-quoted'", re), matches)
	c.Check(match("\"", re), doesntMatch)       // Incomplete string
	c.Check(match("'...'\"...\"", re), matches) // Mixed strings
	c.Check(match("\"...\"", re), matches)
	c.Check(match("`cat file`", re), matches)
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
	msline := NewMkShellLine(G.mk.mklines[0])

	msline.checkShellCommandLine("@# Comment")

	c.Check(s.Output(), equals, "")

	msline.checkShellCommandLine("uname=`uname`; echo $$uname")

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

	msline.checkShellCommandLine("echo ${PKGNAME:Q}") // vucQuotPlain

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: PKGNAME may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${PKGNAME} here.\n")

	msline.checkShellCommandLine("echo \"${CFLAGS:Q}\"") // vucQuotDquot

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please don't use the :Q operator in double quotes.\n"+
		"WARN: fname:1: CFLAGS may not be used in this file.\n"+
		"WARN: fname:1: Please use ${CFLAGS:M*:Q} instead of ${CFLAGS:Q} and make sure the variable appears outside of any quoting characters.\n")

	msline.checkShellCommandLine("echo '${COMMENT:Q}'") // vucQuotSquot

	c.Check(s.Output(), equals, "WARN: fname:1: COMMENT may not be used in this file.\n")

	msline.checkShellCommandLine("echo $$@")

	c.Check(s.Output(), equals, "WARN: fname:1: The $@ shell variable should only be used in double quotes.\n")

	msline.checkShellCommandLine("echo \"$$\"") // As seen by make(1); the shell sees: echo $

	c.Check(s.Output(), equals, "WARN: fname:1: Unquoted $ or strange shell variable found.\n")

	msline.checkShellCommandLine("echo \"\\n\"") // As seen by make(1); the shell sees: echo "\n"

	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"\\\\n\" instead of \"\\n\".\n")
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
	msline := NewMkShellLine(G.mk.mklines[0])

	// foobar="`echo \"foo   bar\"`"
	msline.checkShellCommandLine("foobar=\"`echo \\\"foo   bar\\\"`\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Unknown shell command \"echo\".\n"+
		"ERROR: fname:1: Internal pkglint error: checklineMkShellword state=plain, rest=\"\\\\foo\", shellword=\"\\\\foo\"\n"+
		"ERROR: fname:1: Internal pkglint error: checklineMkShelltext state=continuation rest=\"\\\\\" shellword=\"echo \\\\foo   bar\\\\\"\n")
}

func (s *Suite) TestMkShellLine_CheckShelltext_InternalError2(c *check.C) {
	G.globalData.InitVartypes()
	G.mk = s.NewMkLines("fname",
		"# dummy")
	msline := NewMkShellLine(G.mk.mklines[0])
	s.RegisterTool("pax", "PAX", false)
	G.mk.tools["pax"] = true

	msline.checkShellCommandLine("pax -rwpp -s /.*~$$//g . ${DESTDIR}${PREFIX}")

	c.Check(s.Output(), equals, "ERROR: fname:1: Internal pkglint error: checklineMkShellword state=plain, rest=\"$$//g\", shellword=\"/.*~$$//g\"\n")
}

func (s *Suite) TestChecklineMkShellword(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.InitVartypes()
	msline := NewMkShellLine(NewMkLine(NewLine("fname", 1, "# dummy", nil)))

	c.Check(matches("${list}", `^`+reVarnameDirect+`$`), equals, false)

	msline.checkShellword("${${list}}", false)

	c.Check(s.Output(), equals, "")

	msline.checkShellword("\"$@\"", false)

	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"${.TARGET}\" instead of \"$@\".\n")

	msline.checkShellword("${COMMENT:Q}", true)

	c.Check(s.Output(), equals, "WARN: fname:1: COMMENT may not be used in this file.\n")

	msline.checkShellword("\"${DISTINFO_FILE:Q}\"", true)

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: DISTINFO_FILE may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${DISTINFO_FILE} here.\n")

	msline.checkShellword("embed${DISTINFO_FILE:Q}ded", true)

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: DISTINFO_FILE may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${DISTINFO_FILE} here.\n")
}

func (s *Suite) TestMkShellLine_CheckShellword_InternalError(c *check.C) {
	msline := NewMkShellLine(NewMkLine(NewLine("fname", 1, "# dummy", nil)))

	msline.checkShellword("/.*~$$//g", false)

	c.Check(s.Output(), equals, "ERROR: fname:1: Internal pkglint error: checklineMkShellword state=plain, rest=\"$$//g\", shellword=\"/.*~$$//g\"\n")
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

	msline := NewMkShellLine(NewMkLine(NewLine("Makefile", 3, "# dummy", nil)))

	msline.checkShellCommandLine("for f in *.pl; do ${SED} s,@PREFIX@,${PREFIX}, < $f > $f.tmp && ${MV} $f.tmp $f; done")

	c.Check(s.Output(), equals, "NOTE: Makefile:3: Please use the SUBST framework instead of ${SED} and ${MV}.\n")

	msline.checkShellCommandLine("install -c manpage.1 ${PREFIX}/man/man1/manpage.1")

	c.Check(s.Output(), equals, "WARN: Makefile:3: Please use ${PKGMANDIR} instead of \"man\".\n")

	msline.checkShellCommandLine("cp init-script ${PREFIX}/etc/rc.d/service")

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
