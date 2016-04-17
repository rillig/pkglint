package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestParser_PkgbasePattern(c *check.C) {
	checkRest := func(pattern, expected, rest string) {
		parser := NewParser(dummyLine, pattern)
		actual := parser.PkgbasePattern()
		c.Check(actual, equals, expected)
		c.Check(parser.Rest(), equals, rest)
	}

	checkRest("fltk", "fltk", "")
	checkRest("fltk|", "fltk", "|")
	checkRest("boost-build-1.59.*", "boost-build", "-1.59.*")
	checkRest("${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*")
	checkRest("${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*")
}

func (s *Suite) TestParser_Dependency(c *check.C) {

	checkRest := func(pattern string, expected DependencyPattern, rest string) {
		parser := NewParser(dummyLine, pattern)
		dp := parser.Dependency()
		if c.Check(dp, check.NotNil) {
			c.Check(*dp, equals, expected)
			c.Check(parser.Rest(), equals, rest)
		}
	}
	check := func(pattern string, expected DependencyPattern) {
		checkRest(pattern, expected, "")
	}

	check("fltk>=1.1.5rc1<1.3", DependencyPattern{"fltk", ">=", "1.1.5rc1", "<", "1.3", ""})
	check("libwcalc-1.0*", DependencyPattern{"libwcalc", "", "", "", "", "1.0*"})
	check("${PHP_PKG_PREFIX}-pdo-5.*", DependencyPattern{"${PHP_PKG_PREFIX}-pdo", "", "", "", "", "5.*"})
	check("${PYPKGPREFIX}-metakit-[0-9]*", DependencyPattern{"${PYPKGPREFIX}-metakit", "", "", "", "", "[0-9]*"})
	check("boost-build-1.59.*", DependencyPattern{"boost-build", "", "", "", "", "1.59.*"})
	check("${_EMACS_REQD}", DependencyPattern{"${_EMACS_REQD}", "", "", "", "", ""})
	check("{gcc46,gcc46-libs}>=4.6.0", DependencyPattern{"{gcc46,gcc46-libs}", ">=", "4.6.0", "", "", ""})
	check("perl5-*", DependencyPattern{"perl5", "", "", "", "", "*"})
	check("verilog{,-current}-[0-9]*", DependencyPattern{"verilog{,-current}", "", "", "", "", "[0-9]*"})
	check("mpg123{,-esound,-nas}>=0.59.18", DependencyPattern{"mpg123{,-esound,-nas}", ">=", "0.59.18", "", "", ""})
	check("mysql*-{client,server}-[0-9]*", DependencyPattern{"mysql*-{client,server}", "", "", "", "", "[0-9]*"})
	check("postgresql8[0-35-9]-${module}-[0-9]*", DependencyPattern{"postgresql8[0-35-9]-${module}", "", "", "", "", "[0-9]*"})
	check("ncurses-${NC_VERS}{,nb*}", DependencyPattern{"ncurses", "", "", "", "", "${NC_VERS}{,nb*}"})
	check("xulrunner10>=${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", DependencyPattern{"xulrunner10", ">=", "${MOZ_BRANCH}${MOZ_BRANCH_MINOR}", "", "", ""})
	checkRest("gnome-control-center>=2.20.1{,nb*}", DependencyPattern{"gnome-control-center", ">=", "2.20.1", "", "", ""}, "{,nb*}")
	// "{ssh{,6}-[0-9]*,openssh-[0-9]*}" is not representable using the current data structure
}

func (s *Suite) TestParser_MkTokens(c *check.C) {
	checkRest := func(input string, expectedTokens []*MkToken, expectedRest string) {
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
	check := func(input string, expectedToken *MkToken) {
		checkRest(input, []*MkToken{expectedToken}, "")
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

	check("literal", literal("literal"))
	check("\\/share\\/ { print \"share directory\" }", literal("\\/share\\/ { print \"share directory\" }"))
	check("find . -name \\*.orig -o -name \\*.pre", literal("find . -name \\*.orig -o -name \\*.pre"))
	check("-e 's|\\$${EC2_HOME.*}|EC2_HOME}|g'", literal("-e 's|\\${EC2_HOME.*}|EC2_HOME}|g'"))

	check("${VARIABLE}", varuse("VARIABLE"))
	check("${VARIABLE.param}", varuse("VARIABLE.param"))
	check("${VARIABLE.${param}}", varuse("VARIABLE.${param}"))
	check("${VARIABLE.hicolor-icon-theme}", varuse("VARIABLE.hicolor-icon-theme"))
	check("${VARIABLE.gtk+extra}", varuse("VARIABLE.gtk+extra"))
	check("${VARIABLE:S/old/new/}", varuse("VARIABLE", "S/old/new/"))
	check("${GNUSTEP_LFLAGS:S/-L//g}", varuse("GNUSTEP_LFLAGS", "S/-L//g"))
	check("${SUSE_VERSION:S/.//}", varuse("SUSE_VERSION", "S/.//"))
	check("${MASTER_SITE_GNOME:=sources/alacarte/0.13/}", varuse("MASTER_SITE_GNOME", "=sources/alacarte/0.13/"))
	check("${INCLUDE_DIRS:H:T}", varuse("INCLUDE_DIRS", "H", "T"))
	check("${A.${B.${C.${D}}}}", varuse("A.${B.${C.${D}}}"))
	check("${RUBY_VERSION:C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/}", varuse("RUBY_VERSION", "C/([0-9]+)\\.([0-9]+)\\.([0-9]+)/\\1/"))
	check("${PERL5_${_var_}:Q}", varuse("PERL5_${_var_}", "Q"))
	check("${PKGNAME_REQD:C/(^.*-|^)py([0-9][0-9])-.*/\\2/}", varuse("PKGNAME_REQD", "C/(^.*-|^)py([0-9][0-9])-.*/\\2/"))
	check("${PYLIB:S|/|\\\\/|g}", varuse("PYLIB", "S|/|\\\\/|g"))
	check("${PKGNAME_REQD:C/ruby([0-9][0-9]+)-.*/\\1/}", varuse("PKGNAME_REQD", "C/ruby([0-9][0-9]+)-.*/\\1/"))
	check("${RUBY_SHLIBALIAS:S/\\//\\\\\\//}", varuse("RUBY_SHLIBALIAS", "S/\\//\\\\\\//"))
	check("${RUBY_VER_MAP.${RUBY_VER}:U${RUBY_VER}}", varuse("RUBY_VER_MAP.${RUBY_VER}", "U${RUBY_VER}"))
	check("${RUBY_VER_MAP.${RUBY_VER}:U18}", varuse("RUBY_VER_MAP.${RUBY_VER}", "U18"))
	check("${CONFIGURE_ARGS:S/ENABLE_OSS=no/ENABLE_OSS=yes/g}", varuse("CONFIGURE_ARGS", "S/ENABLE_OSS=no/ENABLE_OSS=yes/g"))
	check("${PLIST_RUBY_DIRS:S,DIR=\"PREFIX/,DIR=\",}", varuse("PLIST_RUBY_DIRS", "S,DIR=\"PREFIX/,DIR=\","))
	check("${LDFLAGS:S/-Wl,//g:Q}", varuse("LDFLAGS", "S/-Wl,//g", "Q"))
	check("${_PERL5_REAL_PACKLIST:S/^/${DESTDIR}/}", varuse("_PERL5_REAL_PACKLIST", "S/^/${DESTDIR}/"))
	check("${_PYTHON_VERSION:C/^([0-9])/\\1./1}", varuse("_PYTHON_VERSION", "C/^([0-9])/\\1./1"))
	check("${PKGNAME:S/py${_PYTHON_VERSION}/py${i}/}", varuse("PKGNAME", "S/py${_PYTHON_VERSION}/py${i}/"))
	check("${PKGNAME:C/-[0-9].*$/-[0-9]*/}", varuse("PKGNAME", "C/-[0-9].*$/-[0-9]*/"))
	check("${PKGNAME:S/py${_PYTHON_VERSION}/py${i}/:C/-[0-9].*$/-[0-9]*/}", varuse("PKGNAME", "S/py${_PYTHON_VERSION}/py${i}/", "C/-[0-9].*$/-[0-9]*/"))
	check("${_PERL5_VARS:tl:S/^/-V:/}", varuse("_PERL5_VARS", "tl", "S/^/-V:/"))
	check("${_PERL5_VARS_OUT:M${_var_:tl}=*:S/^${_var_:tl}=${_PERL5_PREFIX:=/}//}", varuse("_PERL5_VARS_OUT", "M${_var_:tl}=*", "S/^${_var_:tl}=${_PERL5_PREFIX:=/}//"))
	check("${RUBY${RUBY_VER}_PATCHLEVEL}", varuse("RUBY${RUBY_VER}_PATCHLEVEL"))
	check("${DISTFILES:M*.gem}", varuse("DISTFILES", "M*.gem"))
	check("${LOCALBASE:S^/^_^}", varuse("LOCALBASE", "S^/^_^"))
	check("${SOURCES:%.c=%.o}", varuse("SOURCES", "%.c=%.o"))
	check("${GIT_TEMPLATES:@.t.@ ${EGDIR}/${GIT_TEMPLATEDIR}/${.t.} ${PREFIX}/${GIT_CORE_TEMPLATEDIR}/${.t.} @:M*}",
		varuse("GIT_TEMPLATES", "@.t.@ ${EGDIR}/${GIT_TEMPLATEDIR}/${.t.} ${PREFIX}/${GIT_CORE_TEMPLATEDIR}/${.t.} @", "M*"))
	check("${DISTNAME:C:_:-:}", varuse("DISTNAME", "C:_:-:"))
	check("${CF_FILES:H:O:u:S@^@${PKG_SYSCONFDIR}/@}", varuse("CF_FILES", "H", "O", "u", "S@^@${PKG_SYSCONFDIR}/@"))
	check("${ALT_GCC_RTS:S%${LOCALBASE}%%:S%/%%}", varuse("ALT_GCC_RTS", "S%${LOCALBASE}%%", "S%/%%"))
	check("${PREFIX:C;///*;/;g:C;/$;;}", varuse("PREFIX", "C;///*;/;g", "C;/$;;"))
	check("${GZIP_CMD:[1]:Q}", varuse("GZIP_CMD", "[1]", "Q"))
	check("${DISTNAME:C/-[0-9]+$$//:C/_/-/}", varuse("DISTNAME", "C/-[0-9]+$$//", "C/_/-/"))
	check("${DISTNAME:slang%=slang2%}", varuse("DISTNAME", "slang%=slang2%"))
	check("${OSMAP_SUBSTVARS:@v@-e 's,\\@${v}\\@,${${v}},g' @}", varuse("OSMAP_SUBSTVARS", "@v@-e 's,\\@${v}\\@,${${v}},g' @"))
	check("${BRANDELF:D${BRANDELF} -t Linux ${LINUX_LDCONFIG}:U${TRUE}}", varuse("BRANDELF", "D${BRANDELF} -t Linux ${LINUX_LDCONFIG}", "U${TRUE}"))

	/* weird features */
	check("${${EMACS_VERSION_MAJOR}>22:?@comment :}", varuse("${EMACS_VERSION_MAJOR}>22", "?@comment :"))
	check("${empty(CFLAGS):?:-cflags ${CFLAGS:Q}}", varuse("empty(CFLAGS)", "?:-cflags ${CFLAGS:Q}"))

	check("${${XKBBASE}/xkbcomp:L:Q}", varuse("${XKBBASE}/xkbcomp", "L", "Q"))
	check("${${PKGBASE} ${PKGVERSION}:L}", varuse("${PKGBASE} ${PKGVERSION}", "L"))

	check("${${${PKG_INFO} -E ${d} || echo:L:sh}:L:C/[^[0-9]]*/ /g:[1..3]:ts.}",
		varuse("${${PKG_INFO} -E ${d} || echo:L:sh}", "L", "C/[^[0-9]]*/ /g", "[1..3]", "ts."))

	check("${VAR:S/-//S/.//}", varuseText("${VAR:S/-//S/.//}", "VAR", "S/-//", "S/.//")) // For :S and :C, the colon can be left out.

	check("${VAR:ts}", varuse("VAR", "ts"))                 // The separator character can be left out.
	check("${VAR:ts\\000012}", varuse("VAR", "ts\\000012")) // The separator character can be a long octal number.
	check("${VAR:ts\\124}", varuse("VAR", "ts\\124"))       // Or even decimal.

	check("$(GNUSTEP_USER_ROOT)", varuseText("$(GNUSTEP_USER_ROOT)", "GNUSTEP_USER_ROOT"))
	c.Check(s.Output(), equals, "WARN: Please use curly braces {} instead of round parentheses () for GNUSTEP_USER_ROOT.\n")

	checkRest("${VAR)", nil, "${VAR)") // Opening brace, closing parenthesis
	checkRest("$(VAR}", nil, "$(VAR}") // Opening parenthesis, closing brace
	c.Check(s.Output(), equals, "WARN: Please use curly braces {} instead of round parentheses () for VAR.\n")

	check("${PLIST_SUBST_VARS:@var@${var}=${${var}:Q}@}", varuse("PLIST_SUBST_VARS", "@var@${var}=${${var}:Q}@"))
	check("${PLIST_SUBST_VARS:@var@${var}=${${var}:Q}}", varuse("PLIST_SUBST_VARS", "@var@${var}=${${var}:Q}")) // Missing @ at the end
	c.Check(s.Output(), equals, "WARN: Modifier ${PLIST_SUBST_VARS:@var@...@} is missing the final \"@\".\n")

	checkRest("hello, ${W:L:tl}orld", []*MkToken{
		literal("hello, "),
		varuse("W", "L", "tl"),
		literal("orld")}, "")
	checkRest("ftp://${PKGNAME}/ ${MASTER_SITES:=subdir/}", []*MkToken{
		literal("ftp://"),
		varuse("PKGNAME"),
		literal("/ "),
		varuse("MASTER_SITES", "=subdir/")}, "")
}

func (s *Suite) TestParser_MkCond(c *check.C) {
	checkRest := func(input string, expectedTree *Tree, expectedRest string) {
		p := NewParser(dummyLine, input)
		actualTree := p.MkCond()
		c.Check(actualTree, deepEquals, expectedTree)
		c.Check(p.Rest(), equals, expectedRest)
	}
	check := func(input string, expectedTree *Tree) {
		checkRest(input, expectedTree, "")
	}
	varuse := func(varname string, modifiers ...string) MkVarUse {
		return MkVarUse{varname: varname, modifiers: modifiers}
	}

	check("${OPSYS:MNetBSD}",
		NewTree("not", NewTree("empty", varuse("OPSYS", "MNetBSD"))))
	check("defined(VARNAME)",
		NewTree("defined", "VARNAME"))
	check("empty(VARNAME)",
		NewTree("empty", varuse("VARNAME")))
	check("!empty(VARNAME)",
		NewTree("not", NewTree("empty", varuse("VARNAME"))))
	check("!empty(VARNAME:M[yY][eE][sS])",
		NewTree("not", NewTree("empty", varuse("VARNAME", "M[yY][eE][sS]"))))
	check("${VARNAME} != \"Value\"",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	check("${VARNAME:Mi386} != \"Value\"",
		NewTree("compareVarStr", varuse("VARNAME", "Mi386"), "!=", "Value"))
	check("${VARNAME} != Value",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	check("\"${VARNAME}\" != Value",
		NewTree("compareVarStr", varuse("VARNAME"), "!=", "Value"))
	check("(defined(VARNAME))",
		NewTree("defined", "VARNAME"))
	check("exists(/etc/hosts)",
		NewTree("exists", "/etc/hosts"))
	check("exists(${PREFIX}/var)",
		NewTree("exists", "${PREFIX}/var"))
	check("${OPSYS} == \"NetBSD\" || ${OPSYS} == \"OpenBSD\"",
		NewTree("or",
			NewTree("compareVarStr", varuse("OPSYS"), "==", "NetBSD"),
			NewTree("compareVarStr", varuse("OPSYS"), "==", "OpenBSD")))
	check("${OPSYS} == \"NetBSD\" && ${MACHINE_ARCH} == \"i386\"",
		NewTree("and",
			NewTree("compareVarStr", varuse("OPSYS"), "==", "NetBSD"),
			NewTree("compareVarStr", varuse("MACHINE_ARCH"), "==", "i386")))
	check("defined(A) && defined(B) || defined(C) && defined(D)",
		NewTree("or",
			NewTree("and",
				NewTree("defined", "A"),
				NewTree("defined", "B")),
			NewTree("and",
				NewTree("defined", "C"),
				NewTree("defined", "D"))))
	check("${MACHINE_ARCH:Mi386} || ${MACHINE_OPSYS:MNetBSD}",
		NewTree("or",
			NewTree("not", NewTree("empty", varuse("MACHINE_ARCH", "Mi386"))),
			NewTree("not", NewTree("empty", varuse("MACHINE_OPSYS", "MNetBSD")))))

	// Exotic cases
	check("0",
		NewTree("literalNum", "0"))
	check("! ( defined(A)  && empty(VARNAME) )",
		NewTree("not", NewTree("and", NewTree("defined", "A"), NewTree("empty", varuse("VARNAME")))))
	check("${REQD_MAJOR} > ${MAJOR}",
		NewTree("compareVarVar", varuse("REQD_MAJOR"), ">", varuse("MAJOR")))
	check("${OS_VERSION} >= 6.5",
		NewTree("compareVarNum", varuse("OS_VERSION"), ">=", "6.5"))
	check("${OS_VERSION} == 5.3",
		NewTree("compareVarNum", varuse("OS_VERSION"), "==", "5.3"))
	check("!empty(${OS_VARIANT:MIllumos})", // Probably not intended
		NewTree("not", NewTree("empty", varuse("${OS_VARIANT:MIllumos}"))))

	// Errors
	checkRest("!empty(PKG_OPTIONS:Msndfile) || defined(PKG_OPTIONS:Msamplerate)",
		NewTree("not", NewTree("empty", varuse("PKG_OPTIONS", "Msndfile"))),
		" || defined(PKG_OPTIONS:Msamplerate)")
}

func (s *Suite) Test_MkVarUse_Mod(c *check.C) {
	varuse := &MkVarUse{"varname", []string{"Q"}}

	c.Check(varuse.Mod(), equals, ":Q")
}

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
	dquotBackt := func(s string) *ShToken { return token(shtWord, s, shqDquotBackt) }
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
	check("$$var", word("$$var"))
	check("$$var$$var", word("$$var$$var"))
	check("$$var;;",
		word("$$var"),
		token(shtCaseSeparator, ";;", shqPlain))
	check("'single-quoted'",
		q(shqSquot, word("'")),
		q(shqSquot, word("single-quoted")),
		q(shqPlain, word("'")))
	c.Check(checkRest("\"" /* none */), equals, "\"")
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
		dquotBackt("`"),
		dquotBackt("ls"),
		token(shtSpace, " ", shqDquotBackt),
		dquotBackt("-1"),
		token(shtSpace, " ", shqDquotBackt),
		token(shtPipe, "|", shqDquotBackt),
		token(shtSpace, " ", shqDquotBackt),
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

	rest := checkRest("COMMENT=\t\\Make $$$$ fast\"",
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
