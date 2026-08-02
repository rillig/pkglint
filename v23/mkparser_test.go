package pkglint

import (
	"encoding/json"
	"strings"
	"testing"
)

func Test_MkParser_MkCond(t *testing.T) {
	tt := NewTester(t)
	defer tt.Done()

	b := NewMkTokenBuilder()

	cmp := func(left MkCondTerm, op string, right MkCondTerm) *MkCond {
		return &MkCond{Compare: &MkCondCompare{left, op, right}}
	}
	cvar := func(name string, modifiers ...MkExprModifier) MkCondTerm {
		return MkCondTerm{Expr: b.Expr(name, modifiers...)}
	}
	cstr := func(s string) MkCondTerm { return MkCondTerm{Str: s} }
	cnum := func(s string) MkCondTerm { return MkCondTerm{Num: s} }

	termVar := func(varname string, mods ...MkExprModifier) *MkCond {
		return &MkCond{Term: &MkCondTerm{Expr: b.Expr(varname, mods...)}}
	}
	termNum := func(num string) *MkCond {
		return &MkCond{Term: &MkCondTerm{Num: num}}
	}
	termStr := func(str string) *MkCond {
		return &MkCond{Term: &MkCondTerm{Str: str}}
	}

	or := func(args ...*MkCond) *MkCond { return &MkCond{Or: args} }
	and := func(args ...*MkCond) *MkCond { return &MkCond{And: args} }
	not := func(cond *MkCond) *MkCond { return &MkCond{Not: cond} }
	call := func(name string, arg string) *MkCond {
		return &MkCond{Call: &MkCondCall{name, arg}}
	}
	empty := func(varname string, mods ...MkExprModifier) *MkCond {
		return &MkCond{Empty: b.Expr(varname, mods...)}
	}
	defined := func(varname string) *MkCond { return &MkCond{Defined: varname} }
	paren := func(cond *MkCond) *MkCond { return &MkCond{Paren: cond} }

	toJSON := func(obj any) string {
		var sb strings.Builder
		encoder := json.NewEncoder(&sb)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(obj)
		tt.AssertNil(err)
		return sb.String()
	}

	testRest := func(input string, expectedTree *MkCond, expectedRest string) {
		// As of July 2019, p.MkCond does not emit warnings;
		// this is left to MkCondChecker.Check.
		line := tt.NewLine("filename.mk", 1, ".if "+input)
		p := NewMkParser(line, input)
		actualTree := p.MkCond()
		tt.CheckDeepEquals(toJSON(actualTree), toJSON(expectedTree))
		tt.CheckEquals(p.Rest(), expectedRest)
	}
	test := func(input string, expectedTree *MkCond) {
		testRest(input, expectedTree, "")
	}

	test("${OPSYS:MNetBSD}",
		termVar("OPSYS", "MNetBSD"))

	test("defined(VARNAME)",
		defined("VARNAME"))

	test("empty(VARNAME)",
		empty("VARNAME"))

	test("!empty(VARNAME)",
		not(empty("VARNAME")))

	test("!empty(VARNAME:M[yY][eE][sS])",
		not(empty("VARNAME", "M[yY][eE][sS]")))

	// Colons are unescaped at this point because they cannot be mistaken for separators anymore.
	test("!empty(USE_TOOLS:Mautoconf\\:run)",
		not(empty("USE_TOOLS", "Mautoconf:run")))

	test("${VARNAME} != \"Value\"",
		cmp(cvar("VARNAME"), "!=", cstr("Value")))

	test("${VARNAME:Mi386} != \"Value\"",
		cmp(cvar("VARNAME", "Mi386"), "!=", cstr("Value")))

	test("${VARNAME} != Value",
		cmp(cvar("VARNAME"), "!=", cstr("Value")))

	test("\"${VARNAME}\" != Value",
		cmp(cvar("VARNAME"), "!=", cstr("Value")))

	test("${pkg} == \"${name}\"",
		cmp(cvar("pkg"), "==", cvar("name")))

	test("\"${pkg}\" == \"${name}\"",
		cmp(cvar("pkg"), "==", cvar("name")))

	// The right-hand side is not analyzed further to keep the data types simple.
	test("${ABC} == \"${A}B${C}\"",
		cmp(cvar("ABC"), "==", cstr("${A}B${C}")))

	test("${ABC} == \"${A}\\\"${B}\\\\${C}$${shellvar}${D}\"",
		cmp(cvar("ABC"), "==", cstr("${A}\"${B}\\${C}$${shellvar}${D}")))

	test("exists(/etc/hosts)",
		call("exists", "/etc/hosts"))

	test("exists(${PREFIX}/var)",
		call("exists", "${PREFIX}/var"))

	test("${OPSYS} == \"NetBSD\" || ${OPSYS} == \"OpenBSD\"",
		or(
			cmp(cvar("OPSYS"), "==", cstr("NetBSD")),
			cmp(cvar("OPSYS"), "==", cstr("OpenBSD"))))

	test("${OPSYS} == \"NetBSD\" && ${MACHINE_ARCH} == \"i386\"",
		and(
			cmp(cvar("OPSYS"), "==", cstr("NetBSD")),
			cmp(cvar("MACHINE_ARCH"), "==", cstr("i386"))))

	test("defined(A) && defined(B) || defined(C) && defined(D)",
		or(
			and(defined("A"), defined("B")),
			and(defined("C"), defined("D"))))

	test("${MACHINE_ARCH:Mi386} || ${MACHINE_OPSYS:MNetBSD}",
		or(
			termVar("MACHINE_ARCH", "Mi386"),
			termVar("MACHINE_OPSYS", "MNetBSD")))

	test("${VAR} == \"${VAR}suffix\"",
		cmp(cvar("VAR"), "==", cstr("${VAR}suffix")))

	// Exotic cases

	// ".if 0" can be used to skip over a block of code.
	test("0",
		termNum("0"))

	test("0xCAFEBABE",
		termNum("0xCAFEBABE"))

	test("${VAR} == 0xCAFEBABE",
		cmp(cvar("VAR"), "==", cnum("0xCAFEBABE")))

	test("! ( defined(A)  && empty(VARNAME) )",
		not(paren(and(defined("A"), empty("VARNAME")))))

	test("${REQD_MAJOR} > ${MAJOR}",
		cmp(cvar("REQD_MAJOR"), ">", cvar("MAJOR")))

	test("${OS_VERSION} >= 6.5",
		cmp(cvar("OS_VERSION"), ">=", cnum("6.5")))

	test("${OS_VERSION} == 5.3",
		cmp(cvar("OS_VERSION"), "==", cnum("5.3")))

	test("!empty(${OS_VARIANT:MIllumos})", // Probably not intended
		not(empty("${OS_VARIANT:MIllumos}")))

	// There may be whitespace before the parenthesis.
	// See devel/bmake/files/cond.c:/^compare_function/.
	test("defined (VARNAME)",
		defined("VARNAME"))

	test("${\"${PKG_OPTIONS:Moption}\":?--enable-option:--disable-option}",
		termVar("\"${PKG_OPTIONS:Moption}\"", "?--enable-option:--disable-option"))

	// Contrary to most other programming languages, the == operator binds
	// more tightly than the ! operator.
	//
	// See MkCondChecker.checkNotCompare.
	test("!${VAR} == value",
		not(cmp(cvar("VAR"), "==", cstr("value"))))

	// The left-hand side of the comparison can be a quoted string.
	test("\"${VAR}suffix\" == value",
		cmp(cstr("${VAR}suffix"), "==", cstr("value")))

	test("\"${VAR}str\"",
		termStr("${VAR}str"))

	test("commands(show-var)",
		call("commands", "show-var"))

	test("exists(/usr/bin)",
		call("exists", "/usr/bin"))

	test("make(show-var)",
		call("make", "show-var"))

	test("target(do-build)",
		call("target", "do-build"))

	test("(!defined(VAR))",
		paren(not(defined("VAR"))))

	test("(((((!defined(VAR))))))",
		paren(paren(paren(paren(paren(not(defined("VAR"))))))))

	test("(${VAR} == \"value\")",
		paren(cmp(cvar("VAR"), "==", cstr("value"))))

	test("(((((${VAR} == \"value\")))))",
		paren(paren(paren(paren(paren(cmp(cvar("VAR"), "==", cstr("value"))))))))

	test("empty()",
		empty(""))

	// TODO: ok "${q}text${q} == ${VAR}"
	// TODO: fail "text${q} == ${VAR}"
	// TODO: ok "${VAR} == ${q}text${q}"

	// Errors

	testRest("defined()",
		nil,
		"defined()")

	testRest("empty(UNFINISHED",
		nil,
		"empty(UNFINISHED")

	testRest("empty(UNFINISHED:Mpattern",
		nil,
		"empty(UNFINISHED:Mpattern")

	testRest("exists(/$$sys)",
		nil,
		"exists(/$$sys)")

	testRest("exists(/unfinished",
		nil,
		"exists(/unfinished")

	testRest("!empty(PKG_OPTIONS:Msndfile) || defined(PKG_OPTIONS:Msamplerate)",
		not(empty("PKG_OPTIONS", "Msndfile")),
		"|| defined(PKG_OPTIONS:Msamplerate)")

	testRest("${LEFT} &&",
		termVar("LEFT"),
		"&&")

	testRest("\"unfinished string literal",
		nil,
		"\"unfinished string literal")

	// Parsing stops before the variable since the comparison between
	// a variable and a string is one of the smallest building blocks.
	// Letting the ${VAR} through and stopping at the == operator would
	// be misleading.
	//
	// Another possibility would be to fix the unfinished string literal
	// and continue parsing. As of April 2019, the error handling is not
	// robust enough to support this approach; magically fixing parse
	// errors might lead to wrong conclusions and warnings.
	testRest("${VAR} == \"unfinished string literal",
		nil,
		"${VAR} == \"unfinished string literal")

	// A logical not must always be followed by an expression.
	testRest("!<",
		nil,
		"<")

	// Empty parentheses are a syntax error.
	testRest("()",
		nil,
		"()")

	// Unfinished conditions are a syntax error.
	testRest("(${VAR}",
		nil,
		"(${VAR}")

	// Too many closing parentheses are a syntax error.
	testRest("(${VAR}))",
		paren(termVar("VAR")),
		")")

	// The left-hand side of the comparison cannot be an unquoted string literal.
	// These would be rejected by bmake as well.
	testRest("value == \"${VAR}suffix\"",
		nil,
		"value == \"${VAR}suffix\"")

	// Function calls need round parentheses instead of curly braces.
	// As of July 2019, bmake silently accepts this wrong expression
	// and interprets it as !defined(empty{USE_CROSS_COMPILE:M[yY][eE][sS]}),
	// which is always true, except if a variable of this strange name
	// were actually defined.
	testRest("!empty{USE_CROSS_COMPILE:M[yY][eE][sS]}",
		nil,
		"empty{USE_CROSS_COMPILE:M[yY][eE][sS]}")

	testRest("unknown(arg)",
		nil,
		"unknown(arg)")

	// The '!' is consumed by the parser.
	testRest("!",
		nil,
		"")
}

func Test_MkParser_mkCondCompare(t *testing.T) {
	tt := NewTester(t)
	defer tt.Done()

	b := NewMkTokenBuilder()

	mkline := tt.NewMkLine("Makefile", 123, ".if ${PKGPATH} == category/pack.age-3+")
	p := NewMkParser(mkline.Line, mkline.Args())
	cond := p.MkCond()

	tt.CheckEquals(p.Rest(), "")
	tt.CheckDeepEquals(
		cond,
		&MkCond{
			Compare: &MkCondCompare{
				Left:  MkCondTerm{Expr: b.Expr("PKGPATH")},
				Op:    "==",
				Right: MkCondTerm{Str: "category/pack.age-3+"}}})
}

func Test_MkParser_PkgbasePattern(t *testing.T) {
	tests := []struct {
		pattern  string
		want     string
		wantRest string
	}{
		{"fltk", "fltk", ""},
		{"fltk-", "fltk", "-"},
		{"fltk|", "fltk", "|"},
		{"boost-build-1.59.*", "boost-build", "-1.59.*"},
		{"${PHP_PKG_PREFIX}-pdo-5.*", "${PHP_PKG_PREFIX}-pdo", "-5.*"},
		{"${PYPKGPREFIX}-metakit-[0-9]*", "${PYPKGPREFIX}-metakit", "-[0-9]*"},

		{"pkgbase-[0-9]*", "pkgbase", "-[0-9]*"},

		{"pkgbase-client-[0-9]*", "pkgbase-client", "-[0-9]*"},

		{"pkgbase-${VARIANT}-[0-9]*", "pkgbase-${VARIANT}", "-[0-9]*"},

		{"pkgbase-${VERSION}-[0-9]*", "pkgbase", "-${VERSION}-[0-9]*"},

		// This PKGNAME pattern is the one from bsd.pkg.mk.
		// The pattern assumes that the version number does not contain a hyphen,
		// which feels a bit too simple.
		//
		// Since variable substitutions are more common for version numbers
		// than for parts of the package name, pkglint treats the PKGNAME
		// as a version number.
		{"pkgbase-${PKGNAME:C/^.*-//}-[0-9]*", "pkgbase", "-${PKGNAME:C/^.*-//}-[0-9]*"},

		// Using the [a-z] pattern in the package base is only rarely seen in the wild.
		{"pkgbase-[a-z]*-1.0", "pkgbase-[a-z]*", "-1.0"},

		// This is a valid package pattern, but it's more complicated
		// than the patterns pkglint can handle as of January 2019.
		//
		// This pattern doesn't have a single package base, which means it cannot be parsed at all.
		{"{ssh{,6}-[0-9]*,openssh-[0-9]*}", "", "{ssh{,6}-[0-9]*,openssh-[0-9]*}"},
	}
	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			tt := NewTester(t)
			defer tt.Done()

			parser := NewMkParser(nil, test.pattern)

			have := parser.PkgbasePattern()
			haveRest := parser.Rest()
			if have != test.want || haveRest != test.wantRest {
				t.Errorf("want %#v, %#v, have %#v, %#v for %#v",
					test.want, test.wantRest, have, haveRest, test.pattern)
			}
		})
	}
}

func Test_MkParser_isPkgbasePart(t *testing.T) {
	tests := []struct {
		str  string
		want bool
	}{
		{"X11", true},
		{"client", true},
		{"${PKGNAME}", true},
		{"[a-z]", true},
		{"{client,server}", true},

		{"1.2", false},
		{"[0-9]*", false},
		{"{5.[1-7].*,6.[0-9]*}", false},
		{"${PKGVERSION}", false},
		{"${PKGNAME:C/^.*-//}", false},
		{">=1.0", false},
		{"_client", false}, // The combination foo-_client looks strange.
	}
	for _, test := range tests {
		t.Run(test.str, func(t *testing.T) {
			tt := NewTester(t)
			defer tt.Done()

			have := (*MkParser)(nil).isPkgbasePart(test.str)
			if have != test.want {
				t.Errorf("want %#v, have %#v for %#v",
					test.want, have, test.str)
			}
		})
	}
}

func Test_MkCondWalker_Walk(t *testing.T) {
	tt := NewTester(t)
	defer tt.Done()

	mkline := tt.NewMkLine("Makefile", 4, ""+
		".if ${VAR:Mmatch} == ${OTHER} || "+
		"${STR} == Str || "+
		"${VAR} == \"${PRE}text${POST}\" || "+
		"${NUM} == 3 && "+
		"defined(VAR) && "+
		"!exists(file.mk) && "+
		"exists(${FILE}) && "+
		"(((${NONEMPTY})))")
	var events []string

	exprStr := func(expr *MkExpr) string {
		strs := make([]string, 1+len(expr.modifiers))
		strs[0] = expr.varname
		for i, mod := range expr.modifiers {
			strs[1+i] = mod.String()
		}
		return strings.Join(strs, ":")
	}

	addEvent := func(name string, args ...string) {
		events = append(events, sprintf("%15s  %s", name, strings.Join(args, ", ")))
	}

	// XXX: Add callbacks for Or if needed.
	//  A good use case would be to check for unsatisfiable .elif conditions.

	mkline.Cond().Walk(&MkCondCallback{
		func(conds []*MkCond) {
			addEvent("and")
		},
		func(cond *MkCond) {
			addEvent("not")
		},
		func(varname string) {
			addEvent("defined", varname)
		},
		func(expr *MkExpr) {
			addEvent("empty", exprStr(expr))
		},
		func(left *MkCondTerm, op string, right *MkCondTerm) {
			assert(left.Expr != nil)
			switch {
			case right.Expr != nil:
				addEvent("compareExprExpr", exprStr(left.Expr), exprStr(right.Expr))
			case right.Num != "":
				addEvent("compareExprNum", exprStr(left.Expr), right.Num)
			default:
				addEvent("compareExprStr", exprStr(left.Expr), right.Str)
			}
		},
		func(name string, arg string) {
			addEvent("call", name, arg)
		},
		func(cond *MkCond) {
			addEvent("paren")
		},
		func(expr *MkExpr) {
			addEvent("var", exprStr(expr))
		},
		func(expr *MkExpr) {
			addEvent("expr", exprStr(expr))
		}})

	tt.CheckDeepEquals(events, []string{
		"compareExprExpr  VAR:Mmatch, OTHER",
		"           expr  VAR:Mmatch",
		"           expr  OTHER",
		" compareExprStr  STR, Str",
		"           expr  STR",
		" compareExprStr  VAR, ${PRE}text${POST}",
		"           expr  VAR",
		"           expr  PRE",
		"           expr  POST",
		"            and  ",
		" compareExprNum  NUM, 3",
		"           expr  NUM",
		"        defined  VAR",
		"           expr  VAR",
		"            not  ",
		"           call  exists, file.mk",
		"           call  exists, ${FILE}",
		"           expr  FILE",
		"          paren  ",
		"          paren  ",
		"          paren  ",
		"            var  NONEMPTY",
		"           expr  NONEMPTY"})
}

// Ensure that the code works even if none of the callbacks are set.
// This is only for code coverage.
func Test_MkCondWalker_Walk__empty_callbacks(t *testing.T) {
	tt := NewTester(t)
	defer tt.Done()

	mkline := tt.NewMkLine("Makefile", 4, ""+
		".if ${VAR:Mmatch} == ${OTHER} || "+
		"${STR} == Str || "+
		"${VAR} == \"${PRE}text${POST}\" || "+
		"${NUM} == 3 && "+
		"defined(VAR) && "+
		"!exists(file.mk) && "+
		"exists(${FILE}) && "+
		"(((${NONEMPTY})))")

	mkline.Cond().Walk(&MkCondCallback{})
}
