package main

import (
	"gopkg.in/check.v1"
	"netbsd.org/pkglint/regex"
)

func (s *Suite) Test_ShTokenizer_ShAtom(c *check.C) {
	t := s.Init(c)

	// checkRest ensures that the given string is parsed to the expected
	// atoms, and returns the remaining text.
	checkRest := func(s string, expected ...*ShAtom) string {
		p := NewShTokenizer(dummyLine, s, false)
		q := shqPlain
		for _, exp := range expected {
			c.Check(p.ShAtom(q), deepEquals, exp)
			q = exp.Quoting
		}
		return p.Rest()
	}

	// check ensures that the given string is parsed to the expected
	// atoms, and that the text is completely consumed by the parser.
	check := func(str string, expected ...*ShAtom) {
		rest := checkRest(str, expected...)
		c.Check(rest, equals, "")
		t.CheckOutputEmpty()
	}

	token := func(typ ShAtomType, text string) *ShAtom {
		return &ShAtom{typ, text, shqPlain, nil}
	}
	operator := func(s string) *ShAtom { return token(shtOperator, s) }
	comment := func(s string) *ShAtom { return token(shtComment, s) }
	varuse := func(varname string, modifiers ...string) *ShAtom {
		text := "${" + varname
		for _, modifier := range modifiers {
			text += ":" + regex.Compile(`[:\\]`).ReplaceAllString(modifier, "\\\\$1")
		}
		text += "}"
		varuse := &MkVarUse{varname: varname, modifiers: modifiers}
		return &ShAtom{shtVaruse, text, shqPlain, varuse}
	}
	text := func(s string) *ShAtom { return token(shtWord, s) }
	whitespace := func(s string) *ShAtom { return token(shtSpace, s) }

	space := whitespace(" ")
	semicolon := operator(";")
	pipe := operator("|")

	q := func(q ShQuoting, atom *ShAtom) *ShAtom {
		return &ShAtom{atom.Type, atom.MkText, q, atom.data}
	}
	backt := func(atom *ShAtom) *ShAtom { return q(shqBackt, atom) }
	dquot := func(atom *ShAtom) *ShAtom { return q(shqDquot, atom) }
	squot := func(atom *ShAtom) *ShAtom { return q(shqSquot, atom) }
	subsh := func(atom *ShAtom) *ShAtom { return q(shqSubsh, atom) }
	backtDquot := func(atom *ShAtom) *ShAtom { return q(shqBacktDquot, atom) }
	backtSquot := func(atom *ShAtom) *ShAtom { return q(shqBacktSquot, atom) }
	dquotBackt := func(atom *ShAtom) *ShAtom { return q(shqDquotBackt, atom) }
	subshSquot := func(atom *ShAtom) *ShAtom { return q(shqSubshSquot, atom) }
	dquotBacktDquot := func(atom *ShAtom) *ShAtom { return q(shqDquotBacktDquot, atom) }
	dquotBacktSquot := func(atom *ShAtom) *ShAtom { return q(shqDquotBacktSquot, atom) }

	check("" /* none */)

	check("$$var",
		text("$$var"))

	check("$$var$$var",
		text("$$var$$var"))

	check("$$var;;",
		text("$$var"),
		operator(";;"))

	check("'single-quoted'",
		squot(text("'")),
		squot(text("single-quoted")),
		text("'"))

	rest := checkRest("\"" /* none */)
	c.Check(rest, equals, "\"")

	check("$${file%.c}.o",
		text("$${file%.c}.o"))

	check("hello",
		text("hello"))

	check("hello, world",
		text("hello,"),
		space,
		text("world"))

	check("\"",
		dquot(text("\"")))

	check("`",
		backt(text("`")))

	check("`cat fname`",
		backt(text("`")),
		backt(text("cat")),
		backt(space),
		backt(text("fname")),
		text("`"))

	check("hello, \"world\"",
		text("hello,"),
		space,
		dquot(text("\"")),
		dquot(text("world")),
		text("\""))

	check("set -e;",
		text("set"),
		space,
		text("-e"),
		semicolon)

	check("cd ${WRKSRC}/doc/man/man3; PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\";",
		text("cd"),
		space,
		varuse("WRKSRC"),
		text("/doc/man/man3"),
		semicolon,
		space,
		text("PAGES="),
		dquot(text("\"")),
		dquotBackt(text("`")),
		dquotBackt(text("ls")),
		dquotBackt(space),
		dquotBackt(text("-1")),
		dquotBackt(space),
		dquotBackt(operator("|")),
		dquotBackt(space),
		dquotBackt(varuse("SED")),
		dquotBackt(space),
		dquotBackt(text("-e")),
		dquotBackt(space),
		dquotBacktSquot(text("'")),
		dquotBacktSquot(text("s,3qt$$,3,")),
		dquotBackt(text("'")),
		dquot(text("`")),
		text("\""),
		semicolon)

	check("ls -1 | ${SED} -e 's,3qt$$,3,'",
		text("ls"), space, text("-1"), space,
		pipe, space,
		varuse("SED"), space, text("-e"), space,
		squot(text("'")), squot(text("s,3qt$$,3,")), text("'"))

	check("(for PAGE in $$PAGES; do ",
		operator("("),
		text("for"),
		space,
		text("PAGE"),
		space,
		text("in"),
		space,
		text("$$PAGES"),
		semicolon,
		space,
		text("do"),
		space)

	check("    ${ECHO} installing ${DESTDIR}${QTPREFIX}/man/man3/$${PAGE}; ",
		whitespace("    "),
		varuse("ECHO"),
		space,
		text("installing"),
		space,
		varuse("DESTDIR"),
		varuse("QTPREFIX"),
		text("/man/man3/$${PAGE}"),
		semicolon,
		space)

	check("    set - X `head -1 $${PAGE}qt`; ",
		whitespace("    "),
		text("set"),
		space,
		text("-"),
		space,
		text("X"),
		space,
		backt(text("`")),
		backt(text("head")),
		backt(space),
		backt(text("-1")),
		backt(space),
		backt(text("$${PAGE}qt")),
		text("`"),
		semicolon,
		space)

	check("`\"one word\"`",
		backt(text("`")),
		backtDquot(text("\"")),
		backtDquot(text("one word")),
		backt(text("\"")),
		text("`"))

	check("$$var \"$$var\" '$$var' `$$var`",
		text("$$var"),
		space,
		dquot(text("\"")),
		dquot(text("$$var")),
		text("\""),
		space,
		squot(text("'")),
		squot(text("$$var")),
		text("'"),
		space,
		backt(text("`")),
		backt(text("$$var")),
		text("`"))

	check("\"`'echo;echo'`\"",
		dquot(text("\"")),
		dquotBackt(text("`")),
		dquotBacktSquot(text("'")),
		dquotBacktSquot(text("echo;echo")),
		dquotBackt(text("'")),
		dquot(text("`")),
		text("\""))

	check("cat<file",
		text("cat"),
		operator("<"),
		text("file"))

	check("-e \"s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g\"",
		text("-e"),
		space,
		dquot(text("\"")),
		dquot(text("s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g")),
		text("\""))

	check("echo $$, $$- $$/ $$; $$| $$,$$/$$;$$-",
		text("echo"),
		space,
		text("$$,"),
		space,
		text("$$-"),
		space,
		text("$$/"),
		space,
		text("$$"),
		semicolon,
		space,
		text("$$"),
		pipe,
		space,
		text("$$,$$/$$"),
		semicolon,
		text("$$-"))

	rest = checkRest("COMMENT=\t\\Make $$$$ fast\"",
		text("COMMENT="),
		whitespace("\t"),
		text("\\Make"),
		space,
		text("$$$$"),
		space,
		text("fast"))
	c.Check(rest, equals, "\"")

	check("var=`echo;echo|echo&echo||echo&&echo>echo`",
		text("var="),
		backt(text("`")),
		backt(text("echo")),
		backt(semicolon),
		backt(text("echo")),
		backt(operator("|")),
		backt(text("echo")),
		backt(operator("&")),
		backt(text("echo")),
		backt(operator("||")),
		backt(text("echo")),
		backt(operator("&&")),
		backt(text("echo")),
		backt(operator(">")),
		backt(text("echo")),
		text("`"))

	check("# comment",
		comment("# comment"))
	check("no#comment",
		text("no#comment"))
	check("`# comment`continue",
		backt(text("`")),
		backt(comment("# comment")),
		text("`"),
		text("continue"))
	check("`no#comment`continue",
		backt(text("`")),
		backt(text("no#comment")),
		text("`"),
		text("continue"))

	check("var=`tr 'A-Z' 'a-z'`",
		text("var="),
		backt(text("`")),
		backt(text("tr")),
		backt(space),
		backtSquot(text("'")),
		backtSquot(text("A-Z")),
		backt(text("'")),
		backt(space),
		backtSquot(text("'")),
		backtSquot(text("a-z")),
		backt(text("'")),
		text("`"))

	check("var=\"`echo \"\\`echo foo\\`\"`\"",
		text("var="),
		dquot(text("\"")),
		dquotBackt(text("`")),
		dquotBackt(text("echo")),
		dquotBackt(space),
		dquotBacktDquot(text("\"")),
		dquotBacktDquot(text("\\`echo foo\\`")), // One atom, since it doesn't influence parsing.
		dquotBackt(text("\"")),
		dquot(text("`")),
		text("\""))

	check("if cond1; then action1; elif cond2; then action2; else action3; fi",
		text("if"), space, text("cond1"), semicolon, space,
		text("then"), space, text("action1"), semicolon, space,
		text("elif"), space, text("cond2"), semicolon, space,
		text("then"), space, text("action2"), semicolon, space,
		text("else"), space, text("action3"), semicolon, space,
		text("fi"))

	if false {
		check("$$(cat)",
			subsh(text("$$(")),
			subsh(text("cat")),
			text(")"))

		check("$$(cat 'file')",
			subsh(text("$$(")),
			subsh(text("cat")),
			subsh(space),
			subshSquot(text("'")),
			subshSquot(text("file")),
			subsh(text("'")),
			text(")"))
	}
}

func (s *Suite) Test_ShTokenizer_ShAtom__quoting(c *check.C) {
	checkQuotingChange := func(input, expectedOutput string) {
		p := NewShTokenizer(dummyLine, input, false)
		q := shqPlain
		result := ""
		for {
			token := p.ShAtom(q)
			if token == nil {
				break
			}
			result += token.MkText
			if token.Quoting != q {
				q = token.Quoting
				result += "[" + q.String() + "]"
			}
		}
		c.Check(result, equals, expectedOutput)
		c.Check(p.Rest(), equals, "")
	}

	checkQuotingChange("hello, world", "hello, world")
	checkQuotingChange("hello, \"world\"", "hello, \"[d]world\"[plain]")
	checkQuotingChange("1 \"\" 2 '' 3 `` 4", "1 \"[d]\"[plain] 2 '[s]'[plain] 3 `[b]`[plain] 4")
	checkQuotingChange("\"\"", "\"[d]\"[plain]")
	checkQuotingChange("''", "'[s]'[plain]")
	checkQuotingChange("``", "`[b]`[plain]")
	checkQuotingChange("x\"x`x`x\"x'x\"x'", "x\"[d]x`[db]x`[d]x\"[plain]x'[s]x\"x'[plain]")
	checkQuotingChange("x\"x`x'x'x`x\"", "x\"[d]x`[db]x'[dbs]x'[db]x`[d]x\"[plain]")
	checkQuotingChange("x\\\"x\\'x\\`x\\\\", "x\\\"x\\'x\\`x\\\\")
	checkQuotingChange("x\"x\\\"x\\'x\\`x\\\\", "x\"[d]x\\\"x\\'x\\`x\\\\")
	checkQuotingChange("x'x\\\"x\\'x\\`x\\\\", "x'[s]x\\\"x\\'[plain]x\\`x\\\\")
	checkQuotingChange("x`x\\\"x\\'x\\`x\\\\", "x`[b]x\\\"x\\'x\\`x\\\\")
}

func (s *Suite) Test_ShTokenizer_ShToken(c *check.C) {
	t := s.Init(c)

	check := func(str string, expected ...*ShToken) {
		p := NewShTokenizer(dummyLine, str, false)
		for _, exp := range expected {
			c.Check(p.ShToken(), deepEquals, exp)
		}
		c.Check(p.Rest(), equals, "")
		t.CheckOutputEmpty()
	}

	check("",
		nil)

	check("echo",
		NewShToken("echo",
			NewShAtom(shtWord, "echo", shqPlain)))

	check("`cat file`",
		NewShToken("`cat file`",
			NewShAtom(shtWord, "`", shqBackt),
			NewShAtom(shtWord, "cat", shqBackt),
			NewShAtom(shtSpace, " ", shqBackt),
			NewShAtom(shtWord, "file", shqBackt),
			NewShAtom(shtWord, "`", shqPlain)))

	check("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		NewShToken("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
			NewShAtom(shtWord, "PAGES=", shqPlain),
			NewShAtom(shtWord, "\"", shqDquot),
			NewShAtom(shtWord, "`", shqDquotBackt),
			NewShAtom(shtWord, "ls", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "-1", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtOperator, "|", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtomVaruse("${SED}", shqDquotBackt, "SED"),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "-e", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "'", shqDquotBacktSquot),
			NewShAtom(shtWord, "s,3qt$$,3,", shqDquotBacktSquot),
			NewShAtom(shtWord, "'", shqDquotBackt),
			NewShAtom(shtWord, "`", shqDquot),
			NewShAtom(shtWord, "\"", shqPlain)))

	check("echo hello, world",
		NewShToken("echo",
			NewShAtom(shtWord, "echo", shqPlain)),
		NewShToken("hello,",
			NewShAtom(shtWord, "hello,", shqPlain)),
		NewShToken("world",
			NewShAtom(shtWord, "world", shqPlain)))

	check("if cond1; then action1; elif cond2; then action2; else action3; fi",
		NewShToken("if", NewShAtom(shtWord, "if", shqPlain)),
		NewShToken("cond1", NewShAtom(shtWord, "cond1", shqPlain)),
		NewShToken(";", NewShAtom(shtOperator, ";", shqPlain)),
		NewShToken("then", NewShAtom(shtWord, "then", shqPlain)),
		NewShToken("action1", NewShAtom(shtWord, "action1", shqPlain)),
		NewShToken(";", NewShAtom(shtOperator, ";", shqPlain)),
		NewShToken("elif", NewShAtom(shtWord, "elif", shqPlain)),
		NewShToken("cond2", NewShAtom(shtWord, "cond2", shqPlain)),
		NewShToken(";", NewShAtom(shtOperator, ";", shqPlain)),
		NewShToken("then", NewShAtom(shtWord, "then", shqPlain)),
		NewShToken("action2", NewShAtom(shtWord, "action2", shqPlain)),
		NewShToken(";", NewShAtom(shtOperator, ";", shqPlain)),
		NewShToken("else", NewShAtom(shtWord, "else", shqPlain)),
		NewShToken("action3", NewShAtom(shtWord, "action3", shqPlain)),
		NewShToken(";", NewShAtom(shtOperator, ";", shqPlain)),
		NewShToken("fi", NewShAtom(shtWord, "fi", shqPlain)))

	check("PATH=/nonexistent env PATH=${PATH:Q} true",
		NewShToken("PATH=/nonexistent", NewShAtom(shtWord, "PATH=/nonexistent", shqPlain)),
		NewShToken("env", NewShAtom(shtWord, "env", shqPlain)),
		NewShToken("PATH=${PATH:Q}",
			NewShAtom(shtWord, "PATH=", shqPlain),
			NewShAtomVaruse("${PATH:Q}", shqPlain, "PATH", "Q")),
		NewShToken("true", NewShAtom(shtWord, "true", shqPlain)))

	if false { // Don't know how to tokenize this correctly.
		check("id=$$(${AWK} '{print}' < ${WRKSRC}/idfile)",
			NewShToken("id=$$(${AWK} '{print}' < ${WRKSRC}/idfile)",
				NewShAtom(shtWord, "id=", shqPlain),
				NewShAtom(shtWord, "$$(", shqPlain),
				NewShAtomVaruse("${AWK}", shqPlain, "AWK")))
	}
	check("id=`${AWK} '{print}' < ${WRKSRC}/idfile`",
		NewShToken("id=`${AWK} '{print}' < ${WRKSRC}/idfile`",
			NewShAtom(shtWord, "id=", shqPlain),
			NewShAtom(shtWord, "`", shqBackt),
			NewShAtomVaruse("${AWK}", shqBackt, "AWK"),
			NewShAtom(shtSpace, " ", shqBackt),
			NewShAtom(shtWord, "'", shqBacktSquot),
			NewShAtom(shtWord, "{print}", shqBacktSquot),
			NewShAtom(shtWord, "'", shqBackt),
			NewShAtom(shtSpace, " ", shqBackt),
			NewShAtom(shtOperator, "<", shqBackt),
			NewShAtom(shtSpace, " ", shqBackt),
			NewShAtomVaruse("${WRKSRC}", shqBackt, "WRKSRC"),
			NewShAtom(shtWord, "/idfile", shqBackt),
			NewShAtom(shtWord, "`", shqPlain)))
}
