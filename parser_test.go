package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestParser_PkgbasePattern(c *check.C) {
	test := func(pattern, expected, rest string) {
		parser := NewParser(dummyLine, pattern)
		actual := parser.PkgbasePattern()
		c.Check(actual, equals, expected)
		c.Check(parser.Rest(), equals, rest)
	}

	test("fltk", "fltk", "")
	test("fltk|", "fltk", "|")
	test("boost-build-1.59.*", "boost-build", "-1.59.*")
	test("${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*")
	test("${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*")
}

func (s *Suite) TestParser_Dependency(c *check.C) {

	testDependencyRest := func(pattern string, expected DependencyPattern, rest string) {
		parser := NewParser(dummyLine, pattern)
		dp := parser.Dependency()
		if c.Check(dp, check.NotNil) {
			c.Check(*dp, equals, expected)
			c.Check(parser.Rest(), equals, rest)
		}
	}
	testDependency := func(pattern string, expected DependencyPattern) {
		testDependencyRest(pattern, expected, "")
	}

	testDependency("fltk>=1.1.5rc1<1.3", DependencyPattern{"fltk", ">=", "1.1.5rc1", "<", "1.3", ""})
	testDependency("libwcalc-1.0*", DependencyPattern{"libwcalc", "", "", "", "", "1.0*"})
	testDependency("${PHP_PKG_PREFIX}-pdo-5.*", DependencyPattern{"${PHP_PKG_PREFIX}-pdo", "", "", "", "", "5.*"})
	testDependency("${PYPKGPREFIX}-metakit-[0-9]*", DependencyPattern{"${PYPKGPREFIX}-metakit", "", "", "", "", "[0-9]*"})
	testDependency("boost-build-1.59.*", DependencyPattern{"boost-build", "", "", "", "", "1.59.*"})
	testDependency("${_EMACS_REQD}", DependencyPattern{"${_EMACS_REQD}", "", "", "", "", ""})
	testDependency("{gcc46,gcc46-libs}>=4.6.0", DependencyPattern{"{gcc46,gcc46-libs}", ">=", "4.6.0", "", "", ""})
	testDependency("perl5-*", DependencyPattern{"perl5", "", "", "", "", "*"})
	testDependency("verilog{,-current}-[0-9]*", DependencyPattern{"verilog{,-current}", "", "", "", "", "[0-9]*"})
	testDependency("mpg123{,-esound,-nas}>=0.59.18", DependencyPattern{"mpg123{,-esound,-nas}", ">=", "0.59.18", "", "", ""})
	testDependency("mysql*-{client,server}-[0-9]*", DependencyPattern{"mysql*-{client,server}", "", "", "", "", "[0-9]*"})
	testDependency("postgresql8[0-35-9]-${module}-[0-9]*", DependencyPattern{"postgresql8[0-35-9]-${module}", "", "", "", "", "[0-9]*"})
	testDependency("ncurses-${NC_VERS}{,nb*}", DependencyPattern{"ncurses", "", "", "", "", "${NC_VERS}{,nb*}"})
	testDependency("xulrunner10>=${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", DependencyPattern{"xulrunner10", ">=", "${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", "", "", ""})
	testDependencyRest("gnome-control-center>=2.20.1{,nb*}", DependencyPattern{"gnome-control-center", ">=", "2.20.1", "", "", ""}, "{,nb*}")
	// "{ssh{,6}-[0-9]*,openssh-[0-9]*}" is not representable using the current data structure
}

func (s *Suite) TestParser_MkTokens(c *check.C) {
	parse := func(input string, expectedTokens []*MkToken, expectedRest string) {
		p := NewParser(dummyLine, input)
		actualTokens := p.MkTokens()
		c.Check(actualTokens, deepEquals, expectedTokens)
		for i, expectedToken := range expectedTokens {
			if i < len(actualTokens) {
				c.Check(*actualTokens[i], deepEquals, *expectedToken)
			}
		}
		c.Check(p.Rest(), equals, expectedRest)
	}
	token := func(input string, expectedToken *MkToken) {
		parse(input, []*MkToken{expectedToken}, "")
	}
	literal := func(text string) *MkToken {
		return &MkToken{Text: text}
	}
	varuse := func(varname string, modifiers ...string) *MkToken {
		text := "${" + varname
		for _, modifier := range modifiers {
			text += ":" + modifier
		}
		text += "}"
		return &MkToken{Text: text, Varuse: &MkVarUse{varname: varname, modifiers: modifiers}}
	}
	varuseText := func(text, varname string, modifiers ...string) *MkToken {
		return &MkToken{Text: text, Varuse: &MkVarUse{varname: varname, modifiers: modifiers}}
	}

	token("literal", literal("literal"))
	token("\\/share\\/ { print \"share directory\" }", literal("\\/share\\/ { print \"share directory\" }"))
	token("find . -name \\*.orig -o -name \\*.pre", literal("find . -name \\*.orig -o -name \\*.pre"))
	token("-e 's|\\$${EC2_HOME.*}|EC2_HOME}|g'", literal("-e 's|\\${EC2_HOME.*}|EC2_HOME}|g'"))

	token("${VARIABLE}", varuse("VARIABLE"))
	token("${VARIABLE.param}", varuse("VARIABLE.param"))
	token("${VARIABLE.${param}}", varuse("VARIABLE.${param}"))
	token("${VARIABLE.hicolor-icon-theme}", varuse("VARIABLE.hicolor-icon-theme"))
	token("${VARIABLE.gtk+extra}", varuse("VARIABLE.gtk+extra"))
	token("${VARIABLE:S/old/new/}", varuse("VARIABLE", "S/old/new/"))
	token("${GNUSTEP_LFLAGS:S/-L//g}", varuse("GNUSTEP_LFLAGS", "S/-L//g"))
	token("${SUSE_VERSION:S/.//}", varuse("SUSE_VERSION", "S/.//"))
	token("${MASTER_SITE_GNOME:=sources/alacarte/0.13/}", varuse("MASTER_SITE_GNOME", "=sources/alacarte/0.13/"))
	token("${INCLUDE_DIRS:H:T}", varuse("INCLUDE_DIRS", "H", "T"))
	token("${A.${B.${C.${D}}}}", varuse("A.${B.${C.${D}}}"))
	token("${RUBY_VERSION:C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/}", varuse("RUBY_VERSION", "C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/"))
	token("${PERL5_${_var_}:Q}", varuse("PERL5_${_var_}", "Q"))
	token("${PKGNAME_REQD:C/(^.*-|^)py([0-9][0-9])-.*/\\2/}", varuse("PKGNAME_REQD", "C/(^.*-|^)py([0-9][0-9])-.*/\\2/"))
	token("${PYLIB:S|/|\\\\/|g}", varuse("PYLIB", "S|/|\\\\/|g"))
	token("${PKGNAME_REQD:C/ruby([0-9][0-9]+)-.*/\\1/}", varuse("PKGNAME_REQD", "C/ruby([0-9][0-9]+)-.*/\\1/"))
	token("${RUBY_SHLIBALIAS:S/\\//\\\\\\//}", varuse("RUBY_SHLIBALIAS", "S/\\//\\\\\\//"))
	token("${RUBY_VER_MAP.${RUBY_VER}:U${RUBY_VER}}", varuse("RUBY_VER_MAP.${RUBY_VER}", "U${RUBY_VER}"))
	token("${RUBY_VER_MAP.${RUBY_VER}:U18}", varuse("RUBY_VER_MAP.${RUBY_VER}", "U18"))
	token("${CONFIGURE_ARGS:S/ENABLE_OSS=no/ENABLE_OSS=yes/g}", varuse("CONFIGURE_ARGS", "S/ENABLE_OSS=no/ENABLE_OSS=yes/g"))
	token("${PLIST_RUBY_DIRS:S,DIR=\"PREFIX/,DIR=\",}", varuse("PLIST_RUBY_DIRS", "S,DIR=\"PREFIX/,DIR=\","))
	token("${LDFLAGS:S/-Wl,//g:Q}", varuse("LDFLAGS", "S/-Wl,//g", "Q"))
	token("${_PERL5_REAL_PACKLIST:S/^/${DESTDIR}/}", varuse("_PERL5_REAL_PACKLIST", "S/^/${DESTDIR}/"))
	token("${_PYTHON_VERSION:C/^([0-9])/\\1./1}", varuse("_PYTHON_VERSION", "C/^([0-9])/\\1./1"))
	token("${PKGNAME:S/py${_PYTHON_VERSION}/py${i}/}", varuse("PKGNAME", "S/py${_PYTHON_VERSION}/py${i}/"))
	token("${PKGNAME:C/-[0-9].*$/-[0-9]*/}", varuse("PKGNAME", "C/-[0-9].*$/-[0-9]*/"))
	token("${PKGNAME:S/py${_PYTHON_VERSION}/py${i}/:C/-[0-9].*$/-[0-9]*/}", varuse("PKGNAME", "S/py${_PYTHON_VERSION}/py${i}/", "C/-[0-9].*$/-[0-9]*/"))
	token("${_PERL5_VARS:tl:S/^/-V:/}", varuse("_PERL5_VARS", "tl", "S/^/-V:/"))
	token("${_PERL5_VARS_OUT:M${_var_:tl}=*:S/^${_var_:tl}=${_PERL5_PREFIX:=/}//}", varuse("_PERL5_VARS_OUT", "M${_var_:tl}=*", "S/^${_var_:tl}=${_PERL5_PREFIX:=/}//"))
	token("${RUBY${RUBY_VER}_PATCHLEVEL}", varuse("RUBY${RUBY_VER}_PATCHLEVEL"))
	token("${DISTFILES:M*.gem}", varuse("DISTFILES", "M*.gem"))
	token("${LOCALBASE:S^/^_^}", varuse("LOCALBASE", "S^/^_^"))
	token("${SOURCES:%.c=%.o}", varuse("SOURCES", "%.c=%.o"))
	token("${GIT_TEMPLATES:@.t.@ ${EGDIR}/${GIT_TEMPLATEDIR}/${.t.} ${PREFIX}/${GIT_CORE_TEMPLATEDIR}/${.t.} @:M*}",
		varuse("GIT_TEMPLATES", "@.t.@ ${EGDIR}/${GIT_TEMPLATEDIR}/${.t.} ${PREFIX}/${GIT_CORE_TEMPLATEDIR}/${.t.} @", "M*"))
	token("${DISTNAME:C:_:-:}", varuse("DISTNAME", "C:_:-:"))
	token("${CF_FILES:H:O:u:S@^@${PKG_SYSCONFDIR}/@}", varuse("CF_FILES", "H", "O", "u", "S@^@${PKG_SYSCONFDIR}/@"))
	token("${ALT_GCC_RTS:S%${LOCALBASE}%%:S%/%%}", varuse("ALT_GCC_RTS", "S%${LOCALBASE}%%", "S%/%%"))
	token("${PREFIX:C;///*;/;g:C;/$;;}", varuse("PREFIX", "C;///*;/;g", "C;/$;;"))
	token("${GZIP_CMD:[1]:Q}", varuse("GZIP_CMD", "[1]", "Q"))
	token("${DISTNAME:C/-[0-9]+$$//:C/_/-/}", varuse("DISTNAME", "C/-[0-9]+$$//", "C/_/-/"))
	token("${DISTNAME:slang%=slang2%}", varuse("DISTNAME", "slang%=slang2%"))
	token("${OSMAP_SUBSTVARS:@v@-e 's,\\@${v}\\@,${${v}},g' @}", varuse("OSMAP_SUBSTVARS", "@v@-e 's,\\@${v}\\@,${${v}},g' @"))

	/* weird features */
	token("${${EMACS_VERSION_MAJOR}>22:?@comment :}", varuse("${EMACS_VERSION_MAJOR}>22", "?@comment :"))
	token("${empty(CFLAGS):?:-cflags ${CFLAGS:Q}}", varuse("empty(CFLAGS)", "?:-cflags ${CFLAGS:Q}"))

	token("${${XKBBASE}/xkbcomp:L:Q}", varuse("${XKBBASE}/xkbcomp", "L", "Q"))
	token("${${PKGBASE} ${PKGVERSION}:L}", varuse("${PKGBASE} ${PKGVERSION}", "L"))

	token("${${${PKG_INFO} -E ${d} || echo:L:sh}:L:C/[^[0-9]]*/ /g:[1..3]:ts.}",
		varuse("${${PKG_INFO} -E ${d} || echo:L:sh}", "L", "C/[^[0-9]]*/ /g", "[1..3]", "ts."))

	token("${VAR:S/-//S/.//}", varuseText("${VAR:S/-//S/.//}", "VAR", "S/-//", "S/.//")) // For :S and :C, the colon can be left out.

	token("${VAR:ts}", varuse("VAR", "ts"))                 // The separator character can be left out.
	token("${VAR:ts\\000012}", varuse("VAR", "ts\\000012")) // The separator character can be a long octal number.
	token("${VAR:ts\\124}", varuse("VAR", "ts\\124"))       // Or even decimal.

	token("$(GNUSTEP_USER_ROOT)", varuseText("$(GNUSTEP_USER_ROOT)", "GNUSTEP_USER_ROOT"))
	c.Check(s.Output(), equals, "WARN: Please use curly braces {} instead of round parentheses () for GNUSTEP_USER_ROOT.\n")

	parse("${VAR)", nil, "${VAR)") // Opening brace, closing parenthesis
	parse("$(VAR}", nil, "$(VAR}") // Opening parenthesis, closing brace
	c.Check(s.Output(), equals, "WARN: Please use curly braces {} instead of round parentheses () for VAR.\n")

	token("${PLIST_SUBST_VARS:@var@${var}=${${var}:Q}@}", varuse("PLIST_SUBST_VARS", "@var@${var}=${${var}:Q}@"))
	token("${PLIST_SUBST_VARS:@var@${var}=${${var}:Q}}", varuse("PLIST_SUBST_VARS", "@var@${var}=${${var}:Q}")) // Missing @ at the end
	c.Check(s.Output(), equals, "WARN: Modifier ${PLIST_SUBST_VARS:@var@...@} is missing the final \"@\".\n")

	parse("hello, ${W:L:tl}orld", []*MkToken{literal("hello, "), varuse("W", "L", "tl"), literal("orld")}, "")
}

func (s *Suite) TestParser_MkCond(c *check.C) {
	condrest := func(input string, expectedTree *Tree, expectedRest string) {
		p := NewParser(dummyLine, input)
		actualTree := p.MkCond()
		c.Check(actualTree, deepEquals, expectedTree)
		c.Check(p.Rest(), equals, expectedRest)
	}
	cond := func(input string, expectedTree *Tree) {
		condrest(input, expectedTree, "")
	}
	varuse := func(varname string, modifiers ...string) MkVarUse {
		return MkVarUse{varname: varname, modifiers: modifiers}
	}

	cond("${OPSYS:MNetBSD}",
		NewTree("not", NewTree("empty", varuse("OPSYS", "MNetBSD"))))
	cond("defined(VARNAME)",
		NewTree("defined", "VARNAME"))
	cond("empty(VARNAME)",
		NewTree("empty", varuse("VARNAME")))
	cond("!empty(VARNAME)",
		NewTree("not", NewTree("empty", varuse("VARNAME"))))
	cond("!empty(VARNAME:M[yY][eE][sS])",
		NewTree("not", NewTree("empty", varuse("VARNAME", "M[yY][eE][sS]"))))
	cond("${VARNAME} != \"Value\"",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	cond("${VARNAME:Mi386} != \"Value\"",
		NewTree("compareVarStr", varuse("VARNAME", "Mi386"), "!=", "Value"))
	cond("${VARNAME} != Value",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	cond("\"${VARNAME}\" != Value",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	cond("(defined(VARNAME))",
		NewTree("defined", "VARNAME"))
	cond("exists(/etc/hosts)",
		NewTree("exists", "/etc/hosts"))
	cond("exists(${PREFIX}/var)",
		NewTree("exists", "${PREFIX}/var"))
	cond("${OPSYS} == \"NetBSD\" || ${OPSYS} == \"OpenBSD\"",
		NewTree("or",
			NewTree("compareVarStr", varuse("OPSYS"), "==", "NetBSD"),
			NewTree("compareVarStr", varuse("OPSYS"), "==", "OpenBSD")))
	cond("${OPSYS} == \"NetBSD\" && ${MACHINE_ARCH} == \"i386\"",
		NewTree("and",
			NewTree("compareVarStr", varuse("OPSYS"), "==", "NetBSD"),
			NewTree("compareVarStr", varuse("MACHINE_ARCH"), "==", "i386")))
	cond("defined(A) && defined(B) || defined(C) && defined(D)",
		NewTree("or",
			NewTree("and",
				NewTree("defined", "A"),
				NewTree("defined", "B")),
			NewTree("and",
				NewTree("defined", "C"),
				NewTree("defined", "D"))))
	cond("${MACHINE_ARCH:Mi386} || ${MACHINE_OPSYS:MNetBSD}",
		NewTree("or",
			NewTree("not", NewTree("empty", varuse("MACHINE_ARCH", "Mi386"))),
			NewTree("not", NewTree("empty", varuse("MACHINE_OPSYS", "MNetBSD")))))

	// Exotic cases
	cond("0",
		NewTree("literalNum", "0"))
	cond("! ( defined(A)  && empty(VARNAME) )",
		NewTree("not", NewTree("and", NewTree("defined", "A"), NewTree("empty", varuse("VARNAME")))))
	cond("${REQD_MAJOR} > ${MAJOR}",
		NewTree("compareVarVar", varuse("REQD_MAJOR"), ">", varuse("MAJOR")))
	cond("${OS_VERSION} >= 6.5",
		NewTree("compareVarNum", varuse("OS_VERSION"), ">=", "6.5"))
	cond("${OS_VERSION} == 5.3",
		NewTree("compareVarNum", varuse("OS_VERSION"), "==", "5.3"))
	cond("!empty(${OS_VARIANT:MIllumos})", // Probably not intended
		NewTree("not", NewTree("empty", varuse("${OS_VARIANT:MIllumos}"))))

	// Errors
	condrest("!empty(PKG_OPTIONS:Msndfile) || defined(PKG_OPTIONS:Msamplerate)",
		NewTree("not", NewTree("empty", varuse("PKG_OPTIONS", "Msndfile"))),
		" || defined(PKG_OPTIONS:Msamplerate)")
}

func (s *Suite) Test_MkVarUse_Mod(c *check.C) {
	varuse := &MkVarUse{"varname", []string{"Q"}}

	c.Check(varuse.Mod(), equals, ":Q")
}

// @Beta
func (s *Suite) Test_Parser_ShLexeme_Tokens(c *check.C) {
	checkParse := func(s string, expected ...*ShLexeme) {
		p := NewParser(dummyLine, s)
		var q ShQuoting
		for _, exp := range expected {
			c.Check(p.ShLexeme(q), deepEquals, exp)
			q = exp.Quoting
		}
		c.Check(p.Rest(), equals, "")
	}

	lex := func(typ ShLexemeType, text string, quoting ShQuoting) *ShLexeme {
		return &ShLexeme{typ, text, quoting, nil}
	}
	text := func(s string) *ShLexeme { return lex(shlText, s, "") }
	dquot := func(s string) *ShLexeme { return lex(shlText, s, "\"") }
	squot := func(s string) *ShLexeme { return lex(shlText, s, "'") }
	backt := func(s string) *ShLexeme { return lex(shlText, s, "`") }
	dquotBackt := func(s string) *ShLexeme { return lex(shlText, s, "\"`") }
	varuse := func(varname string, modifiers ...string) *ShLexeme {
		text := "${" + varname
		for _, modifier := range modifiers {
			text += ":" + modifier
		}
		text += "}"
		varuse := &MkVarUse{varname: varname, modifiers: modifiers}
		return &ShLexeme{shlVaruse, text, "", varuse}
	}
	q := func(q ShQuoting, lex *ShLexeme) *ShLexeme {
		return &ShLexeme{lex.Type, lex.Text, q, lex.Data}
	}
	whitespace := func(s string) *ShLexeme { return lex(shlSpace, s, "") }
	space := lex(shlSpace, " ", "")
	semicolon := lex(shlSemicolon, ";", "")
	pipe := lex(shlPipe, "|", "")

	checkParse("hello",
		text("hello"))

	checkParse("hello, world",
		text("hello,"),
		space,
		text("world"))

	checkParse("\"",
		dquot("\""))

	checkParse("`",
		backt("`"))

	checkParse("`cat fname`",
		backt("`"),
		backt("cat"),
		lex(shlSpace, " ", "`"),
		backt("fname"),
		text("`"))

	checkParse("hello, \"world\"",
		text("hello,"),
		space,
		dquot("\""),
		dquot("world"),
		text("\""))

	checkParse("set -e;",
		text("set"),
		space,
		text("-e"),
		semicolon)

	checkParse("cd ${WRKSRC}/doc/man/man3; PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\";",
		text("cd"),
		space,
		varuse("WRKSRC"),
		text("/doc/man/man3"),
		semicolon,
		space,
		text("PAGES="),
		dquot("\""),
		dquotBackt("`"),
		dquotBackt("ls"),
		lex(shlSpace, " ", "\"`"),
		dquotBackt("-1"),
		lex(shlSpace, " ", "\"`"),
		lex(shlPipe, "|", "\"`"),
		lex(shlSpace, " ", "\"`"),
		q("\"`", varuse("SED")),
		q("\"`", space),
		q("\"`", text("-e")),
		q("\"`", space),
		q("\"`'", text("'")),
		q("\"`'", text("s,3qt$$,3,")),
		q("\"`", text("'")),
		q("\"", text("`")),
		q("", text("\"")),
		semicolon)

	checkParse("ls -1 | ${SED} -e 's,3qt$$,3,'",
		text("ls"),
		space,
		text("-1"),
		space,
		pipe,
		space,
		varuse("SED"),
		space,
		text("-e"),
		space,
		squot("'"),
		squot("s,3qt$$,3,"),
		text("'"))

	checkParse("(for PAGE in $$PAGES; do ",
		&ShLexeme{shlParenOpen, "(", "", nil},
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

	checkParse("    ${ECHO} installing ${DESTDIR}${QTPREFIX}/man/man3/$${PAGE}; ",
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

	checkParse("    set - X `head -1 $${PAGE}qt`; ",
		whitespace("    "),
		text("set"),
		space,
		text("-"),
		space,
		text("X"),
		space,
		backt("`"),
		backt("head"),
		q("`", space),
		backt("-1"),
		q("`", space),
		backt("$${PAGE}qt"),
		text("`"),
		semicolon,
		space)

	checkParse("`\"one word\"`",
		backt("`"),
		q("`\"", text("\"")),
		q("`\"", text("one word")),
		q("`", text("\"")),
		text("`"))
}

// @Beta
func (s *Suite) Test_Parser_ShLexeme_Quoting(c *check.C) {
	checkQuotingChange := func(input, expectedOutput string) {
		p := NewParser(dummyLine, input)
		var q ShQuoting
		result := ""
		for {
			shlexeme := p.ShLexeme(q)
			if shlexeme == nil {
				break
			}
			result += shlexeme.Text
			if shlexeme.Quoting != q {
				q = shlexeme.Quoting
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
	const (
		B   ShQuoting = "`"
		D             = "\""
		DB            = "\"`"
		DBS           = "\"`'"
	)
	checkParse := func(s string, expected ...*ShWord) {
		p := NewParser(dummyLine, s)
		for _, exp := range expected {
			c.Check(p.ShWord(), deepEquals, exp)
		}
		c.Check(p.Rest(), equals, "")
	}
	lex := func(typ ShLexemeType, s string, q ShQuoting) *ShLexeme {
		return &ShLexeme{typ, s, q, nil}
	}

	checkParse("",
		nil)

	checkParse("echo",
		&ShWord{[]*ShLexeme{
			{shlText, "echo", "", nil}}})

	checkParse("`cat file`",
		&ShWord{[]*ShLexeme{
			{shlText, "`", B, nil},
			{shlText, "cat", B, nil},
			{shlSpace, " ", B, nil},
			{shlText, "file", B, nil},
			{shlText, "`", "", nil}}})

	checkParse("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		&ShWord{[]*ShLexeme{
			{shlText, "PAGES=", "", nil},
			{shlText, "\"", D, nil},
			{shlText, "`", DB, nil},
			{shlText, "ls", DB, nil},
			{shlSpace, " ", DB, nil},
			{shlText, "-1", DB, nil},
			{shlSpace, " ", DB, nil},
			{shlPipe, "|", DB, nil},
			lex(shlSpace, " ", DB),
			{shlVaruse, "${SED}", DB, &MkVarUse{"SED", nil}},
			lex(shlSpace, " ", DB),
			lex(shlText, "-e", DB),
			lex(shlSpace, " ", DB),
			lex(shlText, "'", DBS),
			lex(shlText, "s,3qt$$,3,", DBS),
			lex(shlText, "'", DB),
			lex(shlText, "`", D),
			lex(shlText, "\"", "")}})
}

func (s *Suite) Test_Parser_ShCommand_DataStructures(c *check.C) {
	word := func(lexemes ...*ShLexeme) *ShWord {
		return &ShWord{lexemes}
	}
	plain := func(s string) *ShLexeme {
		return &ShLexeme{shlText, s, "", nil}
	}
	tvaruse := func(s, varname string, modifiers ...string) *ShLexeme {
		return &ShLexeme{shlVaruse, s, "", &MkVarUse{varname, modifiers}}
	}
	plainword := func(s string) *ShWord {
		return &ShWord{[]*ShLexeme{plain(s)}}
	}

	p := NewParser(dummyLine, "PATH=/nonexistent env PATH=${PATH:Q} true")

	shcmd := p.ShCommand()

	expected := &ShCommand{
		[]*ShVarassign{&ShVarassign{"PATH", plainword("/nonexistent")}},
		plainword("env"),
		[]*ShWord{word(plain("PATH="), tvaruse("${PATH:Q}", "PATH", "Q")), plainword("true")}}
	c.Check(shcmd, deepEquals, expected)
	c.Check(shcmd.String(), equals, "ShCommand([ShVarassign(\"PATH\", ShWord([\"/nonexistent\"]))], ShWord([\"env\"]), [ShWord([\"PATH=\" varuse(\"PATH:Q\")]) ShWord([\"true\"])])")
	c.Check(p.Rest(), equals, "")
}

func (s *Suite) Test_Parser_ShCommand_Practical(c *check.C) {
	checkParse := func(cmd, expected string) {
		p := NewParser(dummyLine, cmd)
		shcmd := p.ShCommand()
		if c.Check(shcmd, check.NotNil) {
			c.Check(shcmd.String(), equals, expected)
		}
		c.Check(p.Rest(), equals, "")
	}
	checkParse("echo ${PKGNAME:Q}",
		"ShCommand([], ShWord([\"echo\"]), [ShWord([varuse(\"PKGNAME:Q\")])])")

	checkParse("${ECHO} \"Double-quoted\"",
		"ShCommand([], ShWord([varuse(\"ECHO\")]), [ShWord(["+
			"ShLexeme(text, \"\\\"\", d) "+
			"ShLexeme(text, \"Double-quoted\", d) "+
			"\"\\\"\""+
			"])])")

	checkParse("${ECHO} 'Single-quoted'",
		"ShCommand([], ShWord([varuse(\"ECHO\")]), [ShWord(["+
			"ShLexeme(text, \"'\", s) "+
			"ShLexeme(text, \"Single-quoted\", s) "+
			"\"'\""+
			"])])")

	checkParse("`cat plain`",
		"ShCommand([], ShWord(["+
			"ShLexeme(text, \"`\", b) "+
			"ShLexeme(text, \"cat\", b) "+
			"ShLexeme(space, \" \", b) "+
			"ShLexeme(text, \"plain\", b) "+
			"\"`\""+
			"]), [])")
	checkParse("\"`cat double`\"",
		"ShCommand([], ShWord(["+
			"ShLexeme(text, \"\\\"\", d) "+
			"ShLexeme(text, \"`\", db) "+
			"ShLexeme(text, \"cat\", db) "+
			"ShLexeme(space, \" \", db) "+
			"ShLexeme(text, \"double\", db) "+
			"ShLexeme(text, \"`\", d) "+
			"\"\\\"\""+
			"]), [])")
	checkParse("`\"one word\"`",
		"ShCommand([], ShWord(["+
			"ShLexeme(text, \"`\", b) "+
			"ShLexeme(text, \"\\\"\", bd) "+
			"ShLexeme(text, \"one word\", bd) "+
			"ShLexeme(text, \"\\\"\", b) "+
			"\"`\""+
			"]), [])")

	checkParse("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		"ShCommand([ShVarassign(\"PAGES\", ShWord(["+
			"ShLexeme(text, \"\\\"\", d) "+
			"ShLexeme(text, \"`\", db) "+
			"ShLexeme(text, \"ls\", db) "+
			"ShLexeme(space, \" \", db) "+
			"ShLexeme(text, \"-1\", db) "+
			"ShLexeme(space, \" \", db) "+
			"ShLexeme(pipe, \"|\", db) "+
			"ShLexeme(space, \" \", db) "+
			"varuse(\"SED\") "+
			"ShLexeme(space, \" \", db) "+
			"ShLexeme(text, \"-e\", db) "+
			"ShLexeme(space, \" \", db) "+
			"ShLexeme(text, \"'\", dbs) "+
			"ShLexeme(text, \"s,3qt$$,3,\", dbs) "+
			"ShLexeme(text, \"'\", db) "+
			"ShLexeme(text, \"`\", d) "+
			"\"\\\"\""+
			"]))], <nil>, [])")

	checkParse("var=Plain",
		"ShCommand([ShVarassign(\"var\", ShWord([\"Plain\"]))], <nil>, [])")

	checkParse("var=\"Dquot\"",
		"ShCommand([ShVarassign(\"var\", ShWord(["+
			"ShLexeme(text, \"\\\"\", d) "+
			"ShLexeme(text, \"Dquot\", d) "+
			"\"\\\"\""+
			"]))], <nil>, [])")

	checkParse("var='Squot'",
		"ShCommand([ShVarassign(\"var\", ShWord(["+
			"ShLexeme(text, \"'\", s) "+
			"ShLexeme(text, \"Squot\", s) "+
			"\"'\""+
			"]))], <nil>, [])")

	checkParse("var=Plain\"Dquot\"'Squot'",
		"ShCommand([ShVarassign(\"var\", ShWord(["+
			"\"Plain\" "+
			"ShLexeme(text, \"\\\"\", d) "+
			"ShLexeme(text, \"Dquot\", d) "+
			"\"\\\"\" "+
			"ShLexeme(text, \"'\", s) "+
			"ShLexeme(text, \"Squot\", s) "+
			"\"'\""+
			"]))], <nil>, [])")
}

// @Beta
func (s *Suite) Test_Parser_ShAst(c *check.C) {
	f := func(args ...interface{}) interface{} { return nil }
	Commands := f
	Command := f
	Arg := f
	Varuse := f
	Varassign := f
	Subshell := f
	Pipe := f

	_ = "cd ${WRKSRC}/doc/man/man3; PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\";"

	Commands(
		Command("cd",
			Arg(Varuse("WRKSRC"), "/doc/man/man3")),
		Varassign("PAGES", Subshell(
			Pipe(
				Command("ls", "-1"),
				Command(Varuse("SED"), "-e", "s,3qt$,3,")))))

}
