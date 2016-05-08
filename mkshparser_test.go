package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) Test_MkShParser_Program(c *check.C) {
	parse := func(cmd string, expected *MkShList) {
		p := NewMkShParser(dummyLine, cmd)
		program := p.Program()
		c.Check(program, check.NotNil)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}

	if false {
		parse(""+
			"\tcd ${WRKSRC} && ${FIND} ${${_list_}} -type f ! -name '*.orig' 2>/dev/null "+
			"| pax -rw -pm ${DESTDIR}${PREFIX}/${${_dir_}}",
			NewMkShList())
	}
}

func (s *Suite) Test_MkShParser_List(c *check.C) {

}

func (s *Suite) Test_MkShParser_AndOr(c *check.C) {

}

func (s *Suite) Test_MkShParser_Pipeline(c *check.C) {

}

func (s *Suite) Test_MkShParser_Command(c *check.C) {

}

func (s *Suite) Test_MkShParser_CompoundCommand(c *check.C) {

}

func (s *Suite) Test_MkShParser_Subshell(c *check.C) {

}

func (s *Suite) Test_MkShParser_CompoundList(c *check.C) {

}

func (s *Suite) Test_MkShParser_ForClause(c *check.C) {
	parse := func(cmd string, expected *MkShForClause) {
		p := NewMkShParser(dummyLine, cmd)
		forclause := p.ForClause()
		c.Check(forclause, check.NotNil)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}
	tester := &MkShTester{c}
	params := []*ShToken{tester.Token("\"$@\"")}
	action := tester.ParseCompoundList("action;")

	parse("for var; do action; done",
		&MkShForClause{"var", params, action})
}

func (s *Suite) Test_MkShParser_Wordlist(c *check.C) {

}

func (s *Suite) Test_MkShParser_CaseClause(c *check.C) {

}

func (s *Suite) Test_MkShParser_CaseItem(c *check.C) {

}

func (s *Suite) Test_MkShParser_Pattern(c *check.C) {

}

func (s *Suite) Test_MkShParser_IfClause(c *check.C) {

}

func (s *Suite) Test_MkShParser_WhileClause(c *check.C) {

}

func (s *Suite) Test_MkShParser_UntilClause(c *check.C) {

}

func (s *Suite) Test_MkShParser_FunctionDefinition(c *check.C) {

}

func (s *Suite) Test_MkShParser_BraceGroup(c *check.C) {

}

func (s *Suite) Test_MkShParser_DoGroup(c *check.C) {
	tester := &MkShTester{c}
	check := func(str string, expected *MkShList) {
		p := NewMkShParser(dummyLine, str)
		dogroup := p.DoGroup()
		if c.Check(dogroup, check.NotNil) {
			if !c.Check(dogroup, deepEquals, expected) {
				for i, andor := range dogroup.AndOrs {
					c.Check(andor, deepEquals, expected.AndOrs[i])
				}
			}
		}
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}

	andor := NewMkShAndOr(NewMkShPipeline(false, tester.ParseCommand("action")))
	check("do action; done",
		&MkShList{[]*MkShAndOr{andor}, []MkShSeparator{';'}})
}

func (s *Suite) Test_MkShParser_SimpleCommand(c *check.C) {
	parse := func(cmd string, expected *MkShSimpleCommand) {
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

	fail := func(noncmd string, expectedRest string) {
		p := NewMkShParser(dummyLine, noncmd)
		shcmd := p.SimpleCommand()
		c.Check(shcmd, check.IsNil)
		c.Check(p.tok.parser.Rest(), equals, expectedRest)
		c.Check(s.Output(), equals, "")
	}

	parse("echo ${PKGNAME:Q}",
		NewMkShSimpleCommand(
			NewShToken("echo", NewShAtom(shtWord, "echo", shqPlain)),
			NewShToken("${PKGNAME:Q}", NewShAtomVaruse("${PKGNAME:Q}", shqPlain, "PKGNAME", "Q"))))

	parse("${ECHO} \"Double-quoted\" 'Single-quoted'",
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

	parse("`cat plain` \"`cat double`\" '`cat single`'",
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

	parse("`\"one word\"`",
		NewMkShSimpleCommand(
			NewShToken("`\"one word\"`",
				NewShAtom(shtWord, "`", shqBackt),
				NewShAtom(shtWord, "\"", shqBacktDquot),
				NewShAtom(shtWord, "one word", shqBacktDquot),
				NewShAtom(shtWord, "\"", shqBackt),
				NewShAtom(shtWord, "`", shqPlain))))

	parse("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
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

	parse("var=Plain var=\"Dquot\" var='Squot' var=Plain\"Dquot\"'Squot'",
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

	parse("${RUN} subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"",
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

	parse("PATH=/nonexistent env PATH=${PATH:Q} true",
		NewMkShSimpleCommand(
			NewShToken("PATH=/nonexistent",
				NewShAtom(shtWord, "PATH=/nonexistent", shqPlain)),
			NewShToken("env",
				NewShAtom(shtWord, "env", shqPlain)),
			NewShToken("PATH=${PATH:Q}",
				NewShAtom(shtWord, "PATH=", shqPlain),
				NewShAtomVaruse("${PATH:Q}", shqPlain, "PATH", "Q")),
			NewShToken("true",
				NewShAtom(shtWord, "true", shqPlain))))

	parse("{OpenGrok args",
		NewMkShSimpleCommand(
			NewShToken("{OpenGrok",
				NewShAtom(shtWord, "{OpenGrok", shqPlain)),
			NewShToken("args",
				NewShAtom(shtWord, "args", shqPlain))))

	fail("if clause", "if clause")
	fail("{ group; }", "{ group; }")

}

func (s *Suite) Test_MkShParser_RedirectList(c *check.C) {
}

func (s *Suite) Test_MkShParser_IoRedirect(c *check.C) {
}

func (s *Suite) Test_MkShParser_IoFile(c *check.C) {
}

func (s *Suite) Test_MkShParser_IoHere(c *check.C) {
}

func (s *Suite) Test_MkShParser_NewlineList(c *check.C) {
}

func (s *Suite) Test_MkShParser_Linebreak(c *check.C) {
}

func (s *Suite) Test_MkShParser_SeparatorOp(c *check.C) {

}

func (s *Suite) Test_MkShParser_Separator(c *check.C) {

}

func (s *Suite) Test_MkShParser_SequentialSep(c *check.C) {

}

func (s *Suite) Test_MkShParser_Word(c *check.C) {

}

type MkShTester struct {
	c *check.C
}

func (t *MkShTester) ParseCommand(str string) *MkShCommand {
	p := NewMkShParser(dummyLine, str)
	cmd := p.Command()
	t.c.Check(cmd, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return cmd
}

func (t *MkShTester) ParseSimpleCommand(str string) *MkShSimpleCommand {
	p := NewMkShParser(dummyLine, str)
	parsed := p.SimpleCommand()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}

func (t *MkShTester) ParseCompoundList(str string) *MkShList {
	p := NewMkShParser(dummyLine, str)
	parsed := p.CompoundList()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}

func (t *MkShTester) Token(str string) *ShToken {
	p := NewMkShParser(dummyLine, str)
	parsed := p.peek()
	p.skip()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}
