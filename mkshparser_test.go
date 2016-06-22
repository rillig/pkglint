package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) Test_MkShParser_Program(c *check.C) {
	parse := func(cmd string, expected *MkShList) {
		p := NewMkShParser(dummyLine, cmd, false)
		program := p.Program()
		c.Check(program, deepEquals, expected)
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
	parse := func(cmd string, expected *MkShAndOr) {
		p := NewMkShParser(dummyLine, cmd, false)
		andor := p.AndOr()
		c.Check(andor, deepEquals, expected)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}
	tester := &MkShTester{c}

	parse("simplecmd",
		NewMkShAndOr(NewMkShPipeline(false, tester.ParseCommand("simplecmd"))))

	expected := NewMkShAndOr(NewMkShPipeline(false, tester.ParseCommand("simplecmd1")))
	expected.Add("&&", NewMkShPipeline(false, tester.ParseCommand("simplecmd2")))
	parse("simplecmd1 && simplecmd2", expected)
}

func (s *Suite) Test_MkShParser_Pipeline(c *check.C) {

}

func (s *Suite) Test_MkShParser_Command(c *check.C) {
	parse := func(cmd string, expected *MkShCommand) {
		p := NewMkShParser(dummyLine, cmd, false)
		command := p.Command()
		c.Check(command, deepEquals, expected)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}
	tester := &MkShTester{c}

	parse("simple",
		&MkShCommand{Simple: tester.ParseSimpleCommand("simple")})
}

func (s *Suite) Test_MkShParser_CompoundCommand(c *check.C) {

}

func (s *Suite) Test_MkShParser_Subshell(c *check.C) {

}

func (s *Suite) Test_MkShParser_CompoundList(c *check.C) {
	parse := func(cmd string, expected *MkShList) {
		p := NewMkShParser(dummyLine, cmd, false)
		compoundList := p.CompoundList()
		c.Check(compoundList, deepEquals, expected)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}
	tester := &MkShTester{c}

	parse("simplecmd",
		NewMkShList().AddAndOr(NewMkShAndOr(NewMkShPipeline(false, tester.ParseCommand("simplecmd")))))

	simplecmd1 := NewMkShPipeline(false, tester.ParseCommand("simplecmd1"))
	simplecmd2 := NewMkShPipeline(false, tester.ParseCommand("simplecmd2"))
	expected := NewMkShList().AddAndOr(NewMkShAndOr(simplecmd1).Add("&&", simplecmd2))
	parse("simplecmd1 && simplecmd2", expected)
}

func (s *Suite) Test_MkShParser_ForClause(c *check.C) {
	parse := func(cmd string, expected *MkShForClause) {
		p := NewMkShParser(dummyLine, cmd, false)
		forclause := p.ForClause()
		c.Check(forclause, deepEquals, expected)
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}
	tester := &MkShTester{c}

	params := []*ShToken{tester.Token("\"$$@\"")}
	action := tester.ParseCompoundList("action;")
	parse("for var; do action; done",
		&MkShForClause{"var", params, action})

	abc := []*ShToken{tester.Token("a"), tester.Token("b"), tester.Token("c")}
	parse("for var in a b c; do action; done",
		&MkShForClause{"var", abc, action})

	actions := tester.ParseCompoundList("action1 && action2;")
	parse("for var in a b c; do action1 && action2; done",
		&MkShForClause{"var", abc, actions})
}

func (s *Suite) Test_MkShParser_Wordlist(c *check.C) {

}

func (s *Suite) Test_MkShParser_CaseClause(c *check.C) {
	// TODO: case $var in esac
	// TODO: case $var in pattern) action; esac
	// TODO: case $var in pattern) action;; esac
}

func (s *Suite) Test_MkShParser_CaseItem(c *check.C) {

}

func (s *Suite) Test_MkShParser_Pattern(c *check.C) {

}

func (s *Suite) Test_MkShParser_IfClause(c *check.C) {

}

func (s *Suite) Test_MkShParser_WhileClause(c *check.C) {
	// TODO: while cond; do action; done
}

func (s *Suite) Test_MkShParser_UntilClause(c *check.C) {
	// TODO: until cond; do action; done
}

func (s *Suite) Test_MkShParser_FunctionDefinition(c *check.C) {

}

func (s *Suite) Test_MkShParser_BraceGroup(c *check.C) {

}

func (s *Suite) Test_MkShParser_DoGroup(c *check.C) {
	tester := &MkShTester{c}
	check := func(str string, expected *MkShList) {
		p := NewMkShParser(dummyLine, str, false)
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
		&MkShList{[]*MkShAndOr{andor}, []MkShSeparator{";"}})
}

func (s *Suite) Test_MkShParser_SimpleCommand(c *check.C) {
	parse := func(cmd string, builder *SimpleCommandBuilder) {
		expected := builder.cmd
		p := NewMkShParser(dummyLine, cmd, false)
		shcmd := p.SimpleCommand()
		if c.Check(shcmd, check.NotNil) {
			if !c.Check(shcmd, deepEquals, expected) {
				for i, assignment := range shcmd.Assignments {
					c.Check(assignment, deepEquals, expected.Assignments[i])
				}
				c.Check(shcmd.Name, deepEquals, expected.Name)
				for i, word := range shcmd.Args {
					c.Check(word, deepEquals, expected.Args[i])
				}
				for i, redirection := range shcmd.Redirections {
					c.Check(redirection, deepEquals, expected.Redirections[i])
				}
			}
		}
		c.Check(p.tok.parser.Rest(), equals, "")
		c.Check(s.Output(), equals, "")
	}

	fail := func(noncmd string, expectedRest string) {
		p := NewMkShParser(dummyLine, noncmd, false)
		shcmd := p.SimpleCommand()
		c.Check(shcmd, check.IsNil)
		c.Check(p.tok.parser.Rest(), equals, expectedRest)
		c.Check(s.Output(), equals, "")
	}

	parse("echo ${PKGNAME:Q}",
		NewSimpleCommandBuilder(c).
			Name("echo").
			Arg("${PKGNAME:Q}"))

	parse("${ECHO} \"Double-quoted\" 'Single-quoted'",
		NewSimpleCommandBuilder(c).
			Name("${ECHO}").
			Arg("\"Double-quoted\"").
			Arg("'Single-quoted'"))

	parse("`cat plain` \"`cat double`\" '`cat single`'",
		NewSimpleCommandBuilder(c).
			Name("`cat plain`").
			Arg("\"`cat double`\"").
			Arg("'`cat single`'"))

	parse("`\"one word\"`",
		NewSimpleCommandBuilder(c).
			Name("`\"one word\"`"))

	parse("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		NewSimpleCommandBuilder(c).
			Assignment("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\""))

	parse("var=Plain var=\"Dquot\" var='Squot' var=Plain\"Dquot\"'Squot'",
		NewSimpleCommandBuilder(c).
			Assignment("var=Plain").
			Assignment("var=\"Dquot\"").
			Assignment("var='Squot'").
			Assignment("var=Plain\"Dquot\"'Squot'"))

	parse("${RUN} subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\"",
		NewSimpleCommandBuilder(c).
			Name("${RUN}").
			Arg("subdir=\"`unzip -c \"$$e\" install.rdf | awk '/re/ { print \"hello\" }'`\""))

	parse("PATH=/nonexistent env PATH=${PATH:Q} true",
		NewSimpleCommandBuilder(c).
			Assignment("PATH=/nonexistent").
			Name("env").
			Arg("PATH=${PATH:Q}").
			Arg("true"))

	parse("{OpenGrok args",
		NewSimpleCommandBuilder(c).
			Name("{OpenGrok").
			Arg("args"))

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
	p := NewMkShParser(dummyLine, str, false)
	cmd := p.Command()
	t.c.Check(cmd, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return cmd
}

func (t *MkShTester) ParseSimpleCommand(str string) *MkShSimpleCommand {
	p := NewMkShParser(dummyLine, str, false)
	parsed := p.SimpleCommand()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}

func (t *MkShTester) ParseCompoundList(str string) *MkShList {
	p := NewMkShParser(dummyLine, str, false)
	parsed := p.CompoundList()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}

func (t *MkShTester) Token(str string) *ShToken {
	p := NewMkShParser(dummyLine, str, false)
	parsed := p.peek()
	p.skip()
	t.c.Check(parsed, check.NotNil)
	t.c.Check(p.Rest(), equals, "")
	return parsed
}

type SimpleCommandBuilder struct {
	cmd    *MkShSimpleCommand
	tester *MkShTester
}

func NewSimpleCommandBuilder(c *check.C) *SimpleCommandBuilder {
	cmd := &MkShSimpleCommand{}
	return &SimpleCommandBuilder{cmd, &MkShTester{c}}
}
func (b *SimpleCommandBuilder) Name(name string) *SimpleCommandBuilder {
	b.cmd.Name = b.tester.Token(name)
	return b
}
func (b *SimpleCommandBuilder) Assignment(assignment string) *SimpleCommandBuilder {
	b.cmd.Assignments = append(b.cmd.Assignments, b.tester.Token(assignment))
	return b
}
func (b *SimpleCommandBuilder) Arg(arg string) *SimpleCommandBuilder {
	b.cmd.Args = append(b.cmd.Args, b.tester.Token(arg))
	return b
}
func (b *SimpleCommandBuilder) Redirection(redirection *MkShRedirection) *SimpleCommandBuilder {
	b.cmd.Redirections = append(b.cmd.Redirections, redirection)
	return b
}
func (b *SimpleCommandBuilder) Build() *MkShSimpleCommand {
	return b.cmd
}
