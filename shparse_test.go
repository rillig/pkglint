package main

import (
	check "gopkg.in/check.v1"
)

// @Beta
func (s *Suite) Test_Parser_ShToken_Tokens(c *check.C) {
	checkRest := func(s string, expected ...*ShToken) string {
		p := NewParser(dummyLine, s)
		q := shqPlain
		for _, exp := range expected {
			c.Check(p.ShToken(q), deepEquals, exp)
			q = exp.Quoting
		}
		return p.Rest()
	}
	check := func(s string, expected ...*ShToken) {
		rest := checkRest(s, expected...)
		c.Check(rest, equals, "")
	}

	token := func(typ ShTokenType, text string, quoting ShQuoting) *ShToken {
		return &ShToken{typ, text, quoting, nil}
	}
	word := func(s string) *ShToken { return token(shtWord, s, shqPlain) }
	dquot := func(s string) *ShToken { return token(shtWord, s, shqDquot) }
	squot := func(s string) *ShToken { return token(shtWord, s, shqSquot) }
	backt := func(s string) *ShToken { return token(shtWord, s, shqBackt) }
	varuse := func(varname string, modifiers ...string) *ShToken {
		text := "${" + varname
		for _, modifier := range modifiers {
			text += ":" + modifier
		}
		text += "}"
		varuse := &MkVarUse{varname: varname, modifiers: modifiers}
		return &ShToken{shtVaruse, text, shqPlain, varuse}
	}
	q := func(q ShQuoting, token *ShToken) *ShToken {
		return &ShToken{token.Type, token.Text, q, token.Data}
	}
	whitespace := func(s string) *ShToken { return token(shtSpace, s, shqPlain) }
	space := token(shtSpace, " ", shqPlain)
	semicolon := token(shtSemicolon, ";", shqPlain)
	pipe := token(shtPipe, "|", shqPlain)

	check("" /* none */)

	check("$$var",
		word("$$var"))

	check("$$var$$var",
		word("$$var$$var"))

	check("$$var;;",
		word("$$var"),
		token(shtCaseSeparator, ";;", shqPlain))

	check("'single-quoted'",
		q(shqSquot, word("'")),
		q(shqSquot, word("single-quoted")),
		q(shqPlain, word("'")))

	rest := checkRest("\"" /* none */)
	c.Check(rest, equals, "\"")

	check("$${file%.c}.o",
		word("$${file%.c}.o"))

	check("hello",
		word("hello"))

	check("hello, world",
		word("hello,"),
		space,
		word("world"))

	check("\"",
		dquot("\""))

	check("`",
		backt("`"))

	check("`cat fname`",
		backt("`"),
		backt("cat"),
		token(shtSpace, " ", shqBackt),
		backt("fname"),
		word("`"))

	check("hello, \"world\"",
		word("hello,"),
		space,
		dquot("\""),
		dquot("world"),
		word("\""))

	check("set -e;",
		word("set"),
		space,
		word("-e"),
		semicolon)

	check("cd ${WRKSRC}/doc/man/man3; PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\";",
		word("cd"),
		space,
		varuse("WRKSRC"),
		word("/doc/man/man3"),
		semicolon,
		space,
		word("PAGES="),
		dquot("\""),
		q(shqDquotBackt, word("`")),
		q(shqDquotBackt, word("ls")),
		q(shqDquotBackt, space),
		q(shqDquotBackt, word("-1")),
		q(shqDquotBackt, space),
		token(shtPipe, "|", shqDquotBackt),
		q(shqDquotBackt, space),
		q(shqDquotBackt, varuse("SED")),
		q(shqDquotBackt, space),
		q(shqDquotBackt, word("-e")),
		q(shqDquotBackt, space),
		q(shqDquotBacktSquot, word("'")),
		q(shqDquotBacktSquot, word("s,3qt$$,3,")),
		q(shqDquotBackt, word("'")),
		q(shqDquot, word("`")),
		q(shqPlain, word("\"")),
		semicolon)

	check("ls -1 | ${SED} -e 's,3qt$$,3,'",
		word("ls"),
		space,
		word("-1"),
		space,
		pipe,
		space,
		varuse("SED"),
		space,
		word("-e"),
		space,
		squot("'"),
		squot("s,3qt$$,3,"),
		word("'"))

	check("(for PAGE in $$PAGES; do ",
		&ShToken{shtParenOpen, "(", shqPlain, nil},
		word("for"),
		space,
		word("PAGE"),
		space,
		word("in"),
		space,
		word("$$PAGES"),
		semicolon,
		space,
		word("do"),
		space)

	check("    ${ECHO} installing ${DESTDIR}${QTPREFIX}/man/man3/$${PAGE}; ",
		whitespace("    "),
		varuse("ECHO"),
		space,
		word("installing"),
		space,
		varuse("DESTDIR"),
		varuse("QTPREFIX"),
		word("/man/man3/$${PAGE}"),
		semicolon,
		space)

	check("    set - X `head -1 $${PAGE}qt`; ",
		whitespace("    "),
		word("set"),
		space,
		word("-"),
		space,
		word("X"),
		space,
		backt("`"),
		backt("head"),
		q(shqBackt, space),
		backt("-1"),
		q(shqBackt, space),
		backt("$${PAGE}qt"),
		word("`"),
		semicolon,
		space)

	check("`\"one word\"`",
		backt("`"),
		q(shqBacktDquot, word("\"")),
		q(shqBacktDquot, word("one word")),
		q(shqBackt, word("\"")),
		word("`"))

	check("$$var \"$$var\" '$$var' `$$var`",
		word("$$var"),
		space,
		dquot("\""),
		dquot("$$var"),
		word("\""),
		space,
		squot("'"),
		squot("$$var"),
		word("'"),
		space,
		backt("`"),
		backt("$$var"),
		word("`"))

	check("\"`'echo;echo'`\"",
		q(shqDquot, word("\"")),
		q(shqDquotBackt, word("`")),
		q(shqDquotBacktSquot, word("'")),
		q(shqDquotBacktSquot, word("echo;echo")),
		q(shqDquotBackt, word("'")),
		q(shqDquot, word("`")),
		q(shqPlain, word("\"")))

	check("cat<file",
		word("cat"),
		token(shtRedirect, "<", shqPlain),
		word("file"))

	check("-e \"s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g\"",
		word("-e"),
		space,
		dquot("\""),
		dquot("s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g"),
		word("\""))

	check("echo $$,$$/",
		word("echo"),
		space,
		word("$$,$$/"))

	rest = checkRest("COMMENT=\t\\Make $$$$ fast\"",
		word("COMMENT="),
		whitespace("\t"),
		word("\\Make"),
		space,
		word("$$$$"),
		space,
		word("fast"))
	c.Check(rest, equals, "\"")

	check("var=`echo;echo|echo&echo||echo&&echo>echo`",
		q(shqPlain, word("var=")),
		q(shqBackt, word("`")),
		q(shqBackt, word("echo")),
		q(shqBackt, semicolon),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtPipe, "|", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtBackground, "&", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtOr, "||", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtAnd, "&&", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtRedirect, ">", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqPlain, word("`")))

	check("# comment",
		token(shtComment, "# comment", shqPlain))
	check("no#comment",
		word("no#comment"))
	check("`# comment`continue",
		token(shtWord, "`", shqBackt),
		token(shtComment, "# comment", shqBackt),
		token(shtWord, "`", shqPlain),
		token(shtWord, "continue", shqPlain))
	check("`no#comment`continue",
		token(shtWord, "`", shqBackt),
		token(shtWord, "no#comment", shqBackt),
		token(shtWord, "`", shqPlain),
		token(shtWord, "continue", shqPlain))

	check("var=`tr 'A-Z' 'a-z'`",
		token(shtWord, "var=", shqPlain),
		token(shtWord, "`", shqBackt),
		token(shtWord, "tr", shqBackt),
		token(shtSpace, " ", shqBackt),
		token(shtWord, "'", shqBacktSquot),
		token(shtWord, "A-Z", shqBacktSquot),
		token(shtWord, "'", shqBackt),
		token(shtSpace, " ", shqBackt),
		token(shtWord, "'", shqBacktSquot),
		token(shtWord, "a-z", shqBacktSquot),
		token(shtWord, "'", shqBackt),
		token(shtWord, "`", shqPlain))

	check("var=\"`echo \"\\`echo foo\\`\"`\"",
		token(shtWord, "var=", shqPlain),
		token(shtWord, "\"", shqDquot),
		token(shtWord, "`", shqDquotBackt),
		token(shtWord, "echo", shqDquotBackt),
		token(shtSpace, " ", shqDquotBackt),
		token(shtWord, "\"", shqDquotBacktDquot),
		token(shtWord, "\\`echo foo\\`", shqDquotBacktDquot), // One token, since it doesn’t influence parsing.
		token(shtWord, "\"", shqDquotBackt),
		token(shtWord, "`", shqDquot),
		token(shtWord, "\"", shqPlain))
}

// @Beta
func (s *Suite) Test_Parser_ShToken_Quoting(c *check.C) {
	checkQuotingChange := func(input, expectedOutput string) {
		p := NewParser(dummyLine, input)
		q := shqPlain
		result := ""
		for {
			token := p.ShToken(q)
			if token == nil {
				break
			}
			result += token.Text
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

func (s *Suite) Test_Parser_ShWord(c *check.C) {
	check := func(s string, expected ...*ShWord) {
		p := NewParser(dummyLine, s)
		for _, exp := range expected {
			c.Check(p.ShWord(), deepEquals, exp)
		}
		c.Check(p.Rest(), equals, "")
	}
	token := func(typ ShTokenType, s string, q ShQuoting) *ShToken {
		return &ShToken{typ, s, q, nil}
	}

	check("",
		nil)

	check("echo",
		&ShWord{[]*ShToken{
			{shtWord, "echo", shqPlain, nil}}})

	check("`cat file`",
		&ShWord{[]*ShToken{
			{shtWord, "`", shqBackt, nil},
			{shtWord, "cat", shqBackt, nil},
			{shtSpace, " ", shqBackt, nil},
			{shtWord, "file", shqBackt, nil},
			{shtWord, "`", shqPlain, nil}}})

	check("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		&ShWord{[]*ShToken{
			{shtWord, "PAGES=", shqPlain, nil},
			{shtWord, "\"", shqDquot, nil},
			{shtWord, "`", shqDquotBackt, nil},
			{shtWord, "ls", shqDquotBackt, nil},
			{shtSpace, " ", shqDquotBackt, nil},
			{shtWord, "-1", shqDquotBackt, nil},
			{shtSpace, " ", shqDquotBackt, nil},
			{shtPipe, "|", shqDquotBackt, nil},
			token(shtSpace, " ", shqDquotBackt),
			{shtVaruse, "${SED}", shqDquotBackt, &MkVarUse{"SED", nil}},
			token(shtSpace, " ", shqDquotBackt),
			token(shtWord, "-e", shqDquotBackt),
			token(shtSpace, " ", shqDquotBackt),
			token(shtWord, "'", shqDquotBacktSquot),
			token(shtWord, "s,3qt$$,3,", shqDquotBacktSquot),
			token(shtWord, "'", shqDquotBackt),
			token(shtWord, "`", shqDquot),
			token(shtWord, "\"", shqPlain)}})
}

func (s *Suite) Test_Parser_ShSimpleCmd_DataStructures(c *check.C) {
	word := func(tokens ...*ShToken) *ShWord {
		return &ShWord{tokens}
	}
	plain := func(s string) *ShToken {
		return &ShToken{shtWord, s, shqPlain, nil}
	}
	tvaruse := func(s, varname string, modifiers ...string) *ShToken {
		return &ShToken{shtVaruse, s, shqPlain, &MkVarUse{varname, modifiers}}
	}
	plainword := func(s string) *ShWord {
		return &ShWord{[]*ShToken{plain(s)}}
	}

	p := NewParser(dummyLine, "PATH=/nonexistent env PATH=${PATH:Q} true")

	shcmd := p.ShSimpleCmd()

	expected := &ShSimpleCmd{
		[]*ShVarassign{&ShVarassign{"PATH", plainword("/nonexistent")}},
		plainword("env"),
		[]*ShWord{word(plain("PATH="), tvaruse("${PATH:Q}", "PATH", "Q")), plainword("true")}}
	c.Check(shcmd, deepEquals, expected)
	c.Check(shcmd.String(), equals, "ShSimpleCmd([ShVarassign(\"PATH\", ShWord([\"/nonexistent\"]))], ShWord([\"env\"]), [ShWord([\"PATH=\" varuse(\"PATH:Q\")]) ShWord([\"true\"])])")
	c.Check(p.Rest(), equals, "")
}

func (s *Suite) Test_Parser_ShSimpleCmd_Practical(c *check.C) {
	check := func(cmd, expected string) {
		p := NewParser(dummyLine, cmd)
		shcmd := p.ShSimpleCmd()
		if c.Check(shcmd, check.NotNil) {
			c.Check(shcmd.String(), equals, expected)
		}
		c.Check(p.Rest(), equals, "")
	}

	check("echo ${PKGNAME:Q}",
		"ShSimpleCmd([], ShWord([\"echo\"]), [ShWord([varuse(\"PKGNAME:Q\")])])")

	check("${ECHO} \"Double-quoted\"",
		"ShSimpleCmd([], ShWord([varuse(\"ECHO\")]), [ShWord(["+
			"ShToken(word, \"\\\"\", d) "+
			"ShToken(word, \"Double-quoted\", d) "+
			"\"\\\"\""+
			"])])")

	check("${ECHO} 'Single-quoted'",
		"ShSimpleCmd([], ShWord([varuse(\"ECHO\")]), [ShWord(["+
			"ShToken(word, \"'\", s) "+
			"ShToken(word, \"Single-quoted\", s) "+
			"\"'\""+
			"])])")

	check("`cat plain`",
		"ShSimpleCmd([], ShWord(["+
			"ShToken(word, \"`\", b) "+
			"ShToken(word, \"cat\", b) "+
			"ShToken(space, \" \", b) "+
			"ShToken(word, \"plain\", b) "+
			"\"`\""+
			"]), [])")

	check("\"`cat double`\"",
		"ShSimpleCmd([], ShWord(["+
			"ShToken(word, \"\\\"\", d) "+
			"ShToken(word, \"`\", db) "+
			"ShToken(word, \"cat\", db) "+
			"ShToken(space, \" \", db) "+
			"ShToken(word, \"double\", db) "+
			"ShToken(word, \"`\", d) "+
			"\"\\\"\""+
			"]), [])")

	check("`\"one word\"`",
		"ShSimpleCmd([], ShWord(["+
			"ShToken(word, \"`\", b) "+
			"ShToken(word, \"\\\"\", bd) "+
			"ShToken(word, \"one word\", bd) "+
			"ShToken(word, \"\\\"\", b) "+
			"\"`\""+
			"]), [])")

	check("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		"ShSimpleCmd([ShVarassign(\"PAGES\", ShWord(["+
			"ShToken(word, \"\\\"\", d) "+
			"ShToken(word, \"`\", db) "+
			"ShToken(word, \"ls\", db) "+
			"ShToken(space, \" \", db) "+
			"ShToken(word, \"-1\", db) "+
			"ShToken(space, \" \", db) "+
			"ShToken(pipe, \"|\", db) "+
			"ShToken(space, \" \", db) "+
			"varuse(\"SED\") "+
			"ShToken(space, \" \", db) "+
			"ShToken(word, \"-e\", db) "+
			"ShToken(space, \" \", db) "+
			"ShToken(word, \"'\", dbs) "+
			"ShToken(word, \"s,3qt$$,3,\", dbs) "+
			"ShToken(word, \"'\", db) "+
			"ShToken(word, \"`\", d) "+
			"\"\\\"\""+
			"]))], <nil>, [])")

	check("var=Plain",
		"ShSimpleCmd([ShVarassign(\"var\", ShWord([\"Plain\"]))], <nil>, [])")

	check("var=\"Dquot\"",
		"ShSimpleCmd([ShVarassign(\"var\", ShWord(["+
			"ShToken(word, \"\\\"\", d) "+
			"ShToken(word, \"Dquot\", d) "+
			"\"\\\"\""+
			"]))], <nil>, [])")

	check("var='Squot'",
		"ShSimpleCmd([ShVarassign(\"var\", ShWord(["+
			"ShToken(word, \"'\", s) "+
			"ShToken(word, \"Squot\", s) "+
			"\"'\""+
			"]))], <nil>, [])")

	check("var=Plain\"Dquot\"'Squot'",
		"ShSimpleCmd([ShVarassign(\"var\", ShWord(["+
			"\"Plain\" "+
			"ShToken(word, \"\\\"\", d) "+
			"ShToken(word, \"Dquot\", d) "+
			"\"\\\"\" "+
			"ShToken(word, \"'\", s) "+
			"ShToken(word, \"Squot\", s) "+
			"\"'\""+
			"]))], <nil>, [])")
}
