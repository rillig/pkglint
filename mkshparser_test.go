package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) Test_Parser_ShSimpleCmd_DataStructures(c *check.C) {
	p := NewMkShParser(dummyLine, "PATH=/nonexistent env PATH=${PATH:Q} true")

	shcmd := p.SimpleCommand()

	expected := NewMkShSimpleCommand(
		NewShToken("PATH=/nonexistent",
			NewShAtom(shtWord, "PATH=/nonexistent", shqPlain)),
		NewShToken("env",
			NewShAtom(shtWord, "env", shqPlain)),
		NewShToken("PATH=${PATH:Q}",
			NewShAtom(shtWord, "PATH=", shqPlain),
			NewShAtomVaruse("${PATH:Q}", shqPlain, "PATH", "Q")),
		NewShToken("true",
			NewShAtom(shtWord, "true", shqPlain)))
	c.Check(shcmd, deepEquals, expected)
	c.Check(shcmd.String(), equals, "SimpleCommand(PATH=/nonexistent, env, PATH=${PATH:Q}, true")
	c.Check(p.tok.parser.Rest(), equals, "")
}

func (s *Suite) Test_Parser_ShSimpleCmd(c *check.C) {
	check := func(cmd string, expected *MkShSimpleCommand) {
		p := NewMkShParser(dummyLine, cmd)
		shcmd := p.SimpleCommand()
		if c.Check(shcmd, check.NotNil) {
			if !c.Check(shcmd, deepEquals, expected) {
				for i, word := range shcmd.Words {
					c.Check(word, deepEquals, expected.Words[i])
				}
			}
		}
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}

	check("echo ${PKGNAME:Q}",
		NewMkShSimpleCommand(
			NewShToken("echo", NewShAtom(shtWord, "echo", shqPlain)),
			NewShToken("${PKGNAME:Q}", NewShAtomVaruse("${PKGNAME:Q}", shqPlain, "PKGNAME", "Q"))))

	check("${ECHO} \"Double-quoted\" 'Single-quoted'",
		NewMkShSimpleCommand(
			NewShToken("${ECHO}", NewShAtomVaruse("${ECHO}", shqPlain, "ECHO")),
			NewShToken("\"Double-quoted\"",
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "Double-quoted", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain)),
			NewShToken("'Single-quoted'",
				NewShAtom(shtWord, "'", shqSquot),
				NewShAtom(shtWord, "Single-quoted", shqSquot),
				NewShAtom(shtWord, "'", shqPlain))))

	check("`cat plain` \"`cat double`\" '`cat single`'",
		NewMkShSimpleCommand(
			NewShToken("`cat plain`",
				NewShAtom(shtWord, "`", shqBackt),
				NewShAtom(shtWord, "cat", shqBackt),
				NewShAtom(shtSpace, " ", shqBackt),
				NewShAtom(shtWord, "plain", shqBackt),
				NewShAtom(shtWord, "`", shqPlain)),
			NewShToken("\"`cat double`\"",
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "`", shqDquotBackt),
				NewShAtom(shtWord, "cat", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "double", shqDquotBackt),
				NewShAtom(shtWord, "`", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain)),
			NewShToken("'`cat single`'",
				NewShAtom(shtWord, "'", shqSquot),
				NewShAtom(shtWord, "`cat single`", shqSquot),
				NewShAtom(shtWord, "'", shqPlain))))

	check("`\"one word\"`",
		NewMkShSimpleCommand(
			NewShToken("`\"one word\"`",
				NewShAtom(shtWord, "`", shqBackt),
				NewShAtom(shtWord, "\"", shqBacktDquot),
				NewShAtom(shtWord, "one word", shqBacktDquot),
				NewShAtom(shtWord, "\"", shqBackt),
				NewShAtom(shtWord, "`", shqPlain))))

	check("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		NewMkShSimpleCommand(
			NewShToken("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
				NewShAtom(shtWord, "PAGES=", shqPlain),
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "`", shqDquotBackt),
				NewShAtom(shtWord, "ls", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "-1", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtPipe, "|", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtomVaruse("${SED}", shqDquotBackt, "SED"),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "-e", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "'", shqDquotBacktSquot),
				NewShAtom(shtWord, "s,3qt$$,3,", shqDquotBacktSquot),
				NewShAtom(shtWord, "'", shqDquotBackt),
				NewShAtom(shtWord, "`", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain))))

	check("var=Plain var=\"Dquot\" var='Squot' var=Plain\"Dquot\"'Squot'",
		NewMkShSimpleCommand(
			NewShToken("var=Plain",
				NewShAtom(shtWord, "var=Plain", shqPlain)),
			NewShToken("var=\"Dquot\"",
				NewShAtom(shtWord, "var=", shqPlain),
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "Dquot", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain)),
			NewShToken("var='Squot'",
				NewShAtom(shtWord, "var=", shqPlain),
				NewShAtom(shtWord, "'", shqSquot),
				NewShAtom(shtWord, "Squot", shqSquot),
				NewShAtom(shtWord, "'", shqPlain)),
			NewShToken("var=Plain\"Dquot\"'Squot'",
				NewShAtom(shtWord, "var=Plain", shqPlain),
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "Dquot", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain),
				NewShAtom(shtWord, "'", shqSquot),
				NewShAtom(shtWord, "Squot", shqSquot),
				NewShAtom(shtWord, "'", shqPlain)),
		))

	check("${RUN} subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"",
		NewMkShSimpleCommand(
			NewShToken("${RUN}",
				NewShAtomVaruse("${RUN}", shqPlain, "RUN")),
			NewShToken("subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"",
				NewShAtom(shtWord, "subdir=", shqPlain),
				NewShAtom(shtWord, "\"", shqDquot),
				NewShAtom(shtWord, "`", shqDquotBackt),
				NewShAtom(shtWord, "unzip", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "-c", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "\"", shqDquotBacktDquot),
				NewShAtom(shtWord, "$$e", shqDquotBacktDquot),
				NewShAtom(shtWord, "\"", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "install.rdf", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtPipe, "|", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "awk", shqDquotBackt),
				NewShAtom(shtSpace, " ", shqDquotBackt),
				NewShAtom(shtWord, "'", shqDquotBacktSquot),
				NewShAtom(shtWord, "/re/ { print \"hello\" }", shqDquotBacktSquot),
				NewShAtom(shtWord, "'", shqDquotBackt),
				NewShAtom(shtWord, "`", shqDquot),
				NewShAtom(shtWord, "\"", shqPlain))))
}
